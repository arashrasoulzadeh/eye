package handler

import (
 "context"
 "encoding/json"
 "net/http"
 "strconv"
 "time"
 "sync"
 "sort"
 "log"
 "strings"

 "gorm.io/gorm"
)

type CompressionHandler struct {
 DB *gorm.DB
 mu sync.Mutex
}

func (h *CompressionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
 if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
 tenant := r.URL.Query().Get("tenant_id")
 cutoff, err := strconv.ParseInt(r.URL.Query().Get("to_ts"), 10, 64)
 if tenant == "" || err != nil || cutoff <= 0 || cutoff >= time.Now().UnixMilli() { http.Error(w,"tenant_id and historical to_ts required",400); return }
 if !h.mu.TryLock() { http.Error(w,"assessment already running",429); return }
 defer h.mu.Unlock()
 ctx,cancel := context.WithTimeout(r.Context(),15*time.Second)
 defer cancel()
 db := h.DB.WithContext(ctx)
 var allocated, reusable int64
 var sizeScope string
 var byteExpr string
 switch db.Dialector.Name() {
 case "sqlite":
  var pages, free, pageSize int64
  if db.Raw("PRAGMA page_count").Scan(&pages).Error != nil || db.Raw("PRAGMA freelist_count").Scan(&free).Error != nil || db.Raw("PRAGMA page_size").Scan(&pageSize).Error != nil { http.Error(w,"storage measurement failed",500); return }
  allocated, reusable = pages*pageSize, free*pageSize
  sizeScope = "SQLite main database pages; excludes WAL"
  byteExpr = "length(CAST(COALESCE(payload,'') AS BLOB))"
 case "postgres":
  if db.Raw("SELECT pg_database_size(current_database())").Scan(&allocated).Error != nil { http.Error(w,"storage measurement failed",500); return }
  sizeScope = "PostgreSQL database allocation"
  byteExpr = "octet_length(COALESCE(payload,''))"
 default: http.Error(w,"unsupported database",501); return
 }
 event := r.URL.Query().Get("event_name")
 query := func() *gorm.DB {
  q := db.Table("events").Where("tenant_id = ? AND client_ts <= ?",tenant,cutoff)
  if event != "" { q = q.Where("event_name = ?",event) }
  return q
 }
 var stats struct { Records int64; PayloadBytes int64; PrunedBytes int64 }
 start := time.Now()
 if err := query().Select("COUNT(*) AS records, COALESCE(SUM("+byteExpr+"),0) AS payload_bytes, COUNT(*) * 2 AS pruned_bytes").Scan(&stats).Error; err != nil { http.Error(w,"event assessment failed",500); return }
 scanMS := float64(time.Since(start).Microseconds())/1000
 timings := make([]float64,0,5)
 for i:=0;i<5;i++ {
  var count int64
  started := time.Now()
  if err := query().Count(&count).Error; err != nil { log.Printf("compression count benchmark: %v",err); http.Error(w,"benchmark failed",500); return }
  timings=append(timings,float64(time.Since(started).Microseconds())/1000)
 }
 sort.Float64s(timings)
 inventory := map[string]any{}
 if r.URL.Query().Get("inventory") == "1" { inventory = storageInventory(db,tenant,byteExpr) }
 w.Header().Set("Content-Type","application/json")
 json.NewEncoder(w).Encode(map[string]any{"database_bytes":allocated,"reusable_bytes":reusable,"size_scope":sizeScope,"records":stats.Records,"payload_bytes":stats.PayloadBytes,"pruned_payload_bytes":stats.PrunedBytes,"scan_ms":scanMS,"count_median_ms":timings[2],"count_p95_ms":timings[4],"benchmark_samples":5,"inventory":inventory})
}

type storageGroup struct {
 Name string `json:"name"`
 Records int64 `json:"records"`
 PayloadBytes int64 `json:"payload_bytes"`
 Savings int64 `json:"savings"`
 DimensionBytes int64 `json:"dimension_bytes"`
}
type storageRelation struct {
 Name string `json:"name"`
 TableBytes int64 `json:"table_bytes"`
 IndexBytes int64 `json:"index_bytes"`
}
func storageInventory(db *gorm.DB, tenant, byteExpr string) map[string]any {
 out:=map[string]any{}
 var bounds struct { Records int64 `json:"records"`; Oldest int64 `json:"oldest"`; Newest int64 `json:"newest"` }
 var groups, ages []storageGroup
 var relations []storageRelation
 warnings:=[]string{}
 q:=func()*gorm.DB{return db.Table("events").Where("tenant_id = ?",tenant)}
 if err:=q().Select("COUNT(*) AS records, COALESCE(MIN(client_ts),0) AS oldest, COALESCE(MAX(client_ts),0) AS newest").Scan(&bounds).Error;err!=nil{warnings=append(warnings,"event bounds unavailable")}else{out["bounds"]=bounds}
 savings:="COALESCE(SUM(CASE WHEN "+byteExpr+" > 2 THEN "+byteExpr+" - 2 ELSE 0 END),0)"
 dimExpr:=strings.ReplaceAll(byteExpr,"payload","dims")
 fields:="COALESCE(SUM("+dimExpr+"),0) AS dimension_bytes, COUNT(*) AS records, COALESCE(SUM("+byteExpr+"),0) AS payload_bytes, "+savings+" AS savings"
 if err:=q().Select("event_name AS name, "+fields).Group("event_name").Order("payload_bytes DESC").Limit(100).Scan(&groups).Error;err!=nil{warnings=append(warnings,"event breakdown unavailable")}else{out["events"]=groups}
 now:=time.Now().UnixMilli()
 day:=int64(86400000)
 age:="CASE WHEN client_ts >= ? THEN '0-1 days' WHEN client_ts >= ? THEN '1-7 days' WHEN client_ts >= ? THEN '7-30 days' WHEN client_ts >= ? THEN '30-90 days' ELSE '90+ days' END"
 aged:=q().Select(age+" AS name, payload, dims",now-day,now-7*day,now-30*day,now-90*day)
 if err:=db.Table("(?) AS aged",aged).Select("name, "+fields).Group("name").Scan(&ages).Error;err!=nil{warnings=append(warnings,"age breakdown unavailable")}else{out["ages"]=ages}
 if db.Dialector.Name()=="postgres" {
  if err:=db.Raw("SELECT schemaname || '.' || relname AS name, pg_table_size(relid) AS table_bytes, pg_indexes_size(relid) AS index_bytes FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC LIMIT 100").Scan(&relations).Error;err!=nil{warnings=append(warnings,"relation sizes unavailable")}else{out["relations"]=relations}
  var indexes []struct {
 Name string `json:"name"`; Relation string `json:"relation"`; Bytes int64 `json:"bytes"`; Scans int64 `json:"scans"`; Unique bool `json:"unique"`; Definition string `json:"definition"`
 }
 if err:=db.Raw("SELECT s.schemaname || '.' || s.indexrelname AS name, s.schemaname || '.' || s.relname AS relation, pg_relation_size(s.indexrelid) AS bytes, COALESCE(s.idx_scan,0) AS scans, i.indisunique AS \"unique\", pg_get_indexdef(s.indexrelid) AS definition FROM pg_stat_user_indexes s JOIN pg_index i ON i.indexrelid=s.indexrelid ORDER BY pg_relation_size(s.indexrelid) DESC LIMIT 100").Scan(&indexes).Error;err==nil{out["indexes"]=indexes}else{warnings=append(warnings,"index diagnostics unavailable")}
 var duplicates []struct { First string `json:"first"`; Second string `json:"second"`; Relation string `json:"relation"` }
 if err:=db.Raw("SELECT a.indexrelid::regclass::text AS first, b.indexrelid::regclass::text AS second, a.indrelid::regclass::text AS relation FROM pg_index a JOIN pg_index b ON a.indrelid=b.indrelid AND a.indexrelid<b.indexrelid JOIN pg_class ca ON ca.oid=a.indexrelid JOIN pg_class cb ON cb.oid=b.indexrelid WHERE NOT a.indisunique AND NOT b.indisunique AND a.indisvalid AND b.indisvalid AND ca.relam=cb.relam AND a.indkey=b.indkey AND a.indclass=b.indclass AND a.indcollation=b.indcollation AND a.indoption=b.indoption AND a.indnkeyatts=b.indnkeyatts AND a.indexprs IS NULL AND b.indexprs IS NULL AND a.indpred IS NULL AND b.indpred IS NULL LIMIT 100").Scan(&duplicates).Error;err==nil{out["duplicate_indexes"]=duplicates}else{warnings=append(warnings,"duplicate index diagnostics unavailable")}
 var extension string
  if err:=db.Raw("SELECT COALESCE(MAX(extversion),'') FROM pg_extension WHERE extname='timescaledb'").Scan(&extension).Error;err==nil {
   out["timescale_version"]=extension
   if extension!="" {
    var chunks struct { Total int64 `json:"total"`; Compressed int64 `json:"compressed"`; OlderUncompressed int64 `json:"older_uncompressed"` }
    if err:=db.Raw("SELECT COUNT(*) AS total, COUNT(*) FILTER (WHERE is_compressed) AS compressed, COUNT(*) FILTER (WHERE NOT is_compressed AND range_end < now() - interval '7 days') AS older_uncompressed FROM timescaledb_information.chunks WHERE hypertable_name='events' AND hypertable_schema=current_schema()").Scan(&chunks).Error;err==nil{out["chunks"]=chunks}else{warnings=append(warnings,"Timescale chunk inventory unavailable")}
   }
  }
 } else {
  if err:=db.Raw("SELECT name, SUM(pgsize) AS table_bytes, 0 AS index_bytes FROM dbstat GROUP BY name ORDER BY table_bytes DESC LIMIT 100").Scan(&relations).Error;err!=nil{warnings=append(warnings,"SQLite dbstat unavailable")}else{out["relations"]=relations}
 }
 out["warnings"]=warnings
 return out
}
