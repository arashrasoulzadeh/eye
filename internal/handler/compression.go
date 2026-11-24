package handler

import (
 "context"
 "encoding/json"
 "net/http"
 "strconv"
 "time"
 "sync"

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
 query := db.Table("events").Where("tenant_id = ? AND client_ts <= ?",tenant,cutoff)
 if event := r.URL.Query().Get("event_name"); event != "" { query = query.Where("event_name = ?",event) }
 var stats struct { Records int64; PayloadBytes int64; PrunedBytes int64 }
 start := time.Now()
 if err := query.Select("COUNT(*) AS records, COALESCE(SUM("+byteExpr+"),0) AS payload_bytes, COUNT(*) * 2 AS pruned_bytes").Scan(&stats).Error; err != nil { http.Error(w,"event assessment failed",500); return }
 w.Header().Set("Content-Type","application/json")
 json.NewEncoder(w).Encode(map[string]any{"database_bytes":allocated,"reusable_bytes":reusable,"size_scope":sizeScope,"records":stats.Records,"payload_bytes":stats.PayloadBytes,"pruned_payload_bytes":stats.PrunedBytes,"scan_ms":float64(time.Since(start).Microseconds())/1000})
}
