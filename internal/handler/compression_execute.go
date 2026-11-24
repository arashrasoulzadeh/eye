package handler

import (
 "context"
 "encoding/json"
 "net/http"
 "time"
 "log"
 "sync"

 "gorm.io/gorm"
 "github.com/arashrasoulzadeh/eye/internal/model"
 "github.com/arashrasoulzadeh/eye/internal/service"
)

type CompressionExecuteHandler struct { DB *gorm.DB; mu sync.Mutex }

func (h *CompressionExecuteHandler) ServeHTTP(w http.ResponseWriter,r *http.Request) {
 if r.Method!=http.MethodPost {w.WriteHeader(405);return}
 var p struct { Tenant string `json:"tenant_id"`; Event string `json:"event_name"`; Cutoff int64 `json:"to_ts"`; Confirm string `json:"confirm"`; Action string `json:"action"` }
 if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&p)!=nil {http.Error(w,"invalid request",400);return}
 if !h.mu.TryLock(){http.Error(w,"storage operation running",429);return};defer h.mu.Unlock()
 if p.Action=="inference_run_retention" {h.inferenceRunRetention(w,r,p.Tenant,p.Cutoff,p.Confirm);return}
 if p.Action=="rowstore" {h.rowstore(w,r,p.Confirm);return}
 if p.Action=="columnstore" {h.columnstore(w,r,p.Confirm);return}
 if (p.Action!="" && p.Action!="prune" && p.Action!="compact") || p.Tenant=="" || p.Event=="" || p.Cutoff<=0 || p.Cutoff>time.Now().Add(-24*time.Hour).UnixMilli() || (p.Action!="compact" && p.Confirm!="PRUNE "+p.Tenant+" "+p.Event) || (p.Action=="compact" && p.Confirm!="COMPACT "+p.Tenant+" "+p.Event) {http.Error(w,"explicit tenant, event, historical cutoff and confirmation required",400);return}
 ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
 if p.Action=="compact" {
  cutoff:=p.Cutoff-p.Cutoff%3600000
  n,err:=service.CompactEventBatch(ctx,h.DB,p.Tenant,p.Event,cutoff,500)
  if err!=nil {http.Error(w,"compaction failed",500);return}
  w.Header().Set("Content-Type","application/json")
  json.NewEncoder(w).Encode(map[string]any{"updated":n,"cutoff":cutoff})
  return
 }
 var before,after int64
 var removed int64
 started:=time.Now()
 err:=h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  var rows []model.Event
  if err:=tx.Where("tenant_id = ? AND event_name = ? AND client_ts <= ? AND payload <> ?",p.Tenant,p.Event,p.Cutoff,"{}").Order("client_ts,event_id").Limit(500).Find(&rows).Error;err!=nil{return err}
  for _,row:=range rows {
   res:=tx.Model(&model.Event{}).Where("tenant_id = ? AND event_name = ? AND event_id = ? AND ingest_ts = ? AND payload = ?",p.Tenant,p.Event,row.EventID,row.IngestTS,row.Payload).Update("payload","{}")
   if res.Error!=nil{return res.Error}
   if res.RowsAffected==1 {before+=int64(len(row.Payload));after+=2;removed++}
  }
  return nil
 })
 if err!=nil {http.Error(w,"pruning failed; transaction rolled back",500);return}
 w.Header().Set("Content-Type","application/json")
 json.NewEncoder(w).Encode(map[string]any{"updated":removed,"before_payload_bytes":before,"after_payload_bytes":after,"duration_ms":float64(time.Since(started).Microseconds())/1000})
}

func (h *CompressionExecuteHandler) columnstore(w http.ResponseWriter,r *http.Request,confirm string) {
 if confirm!="COMPRESS GLOBAL EVENTS" {http.Error(w,"global confirmation required",400);return}
 if h.DB.Dialector.Name()!="postgres" {http.Error(w,"TimescaleDB required",400);return}
 ctx,cancel:=context.WithTimeout(r.Context(),120*time.Second);defer cancel()
 started:=time.Now()
 var chunk string
 var before,after int64
 var beforeCount,afterCount int64
 var beforeMS,afterMS float64
 err:=h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  var acquired bool
  if err:=tx.Raw("SELECT pg_try_advisory_xact_lock(hashtextextended('eye:events:columnstore',0))").Scan(&acquired).Error;err!=nil{return err}
  if !acquired{return gorm.ErrInvalidTransaction}
  if err:=tx.Exec("SET LOCAL lock_timeout = '3s'").Error;err!=nil{return err}
  if err:=tx.Exec("SET LOCAL statement_timeout = '110s'").Error;err!=nil{return err}
  if err:=tx.Raw("SELECT format('%I.%I',chunk_schema,chunk_name) FROM timescaledb_information.chunks WHERE hypertable_schema=current_schema() AND hypertable_name='events' AND NOT is_compressed AND range_end < now()-interval '7 days' ORDER BY range_end LIMIT 1").Scan(&chunk).Error;err!=nil{return err}
  if chunk=="" {return nil}
  if err:=tx.Raw("SELECT hypertable_size('events')").Scan(&before).Error;err!=nil{return err}
  t:=time.Now()
  if err:=tx.Raw("SELECT COUNT(*) FROM "+chunk).Scan(&beforeCount).Error;err!=nil{return err}
  beforeMS=float64(time.Since(t).Microseconds())/1000
  var enabled bool
  if err:=tx.Raw("SELECT compression_enabled FROM timescaledb_information.hypertables WHERE hypertable_schema=current_schema() AND hypertable_name='events'").Scan(&enabled).Error;err!=nil{return err}
  if !enabled {
   if err:=tx.Exec("ALTER TABLE events SET (timescaledb.enable_columnstore=true, timescaledb.segmentby='tenant_id,event_name', timescaledb.orderby='ingest_ts DESC')").Error;err!=nil{return err}
  }
  if err:=tx.Exec("CALL convert_to_columnstore(CAST(? AS regclass))",chunk).Error;err!=nil{return err}
  t=time.Now()
  if err:=tx.Raw("SELECT COUNT(*) FROM "+chunk).Scan(&afterCount).Error;err!=nil{return err}
  afterMS=float64(time.Since(t).Microseconds())/1000
  if beforeCount!=afterCount{return gorm.ErrInvalidData}
  return tx.Raw("SELECT hypertable_size('events')").Scan(&after).Error
 })
 if err!=nil {log.Printf("columnstore compression: %v",err);http.Error(w,"columnstore operation failed; see server logs",500);return}
 w.Header().Set("Content-Type","application/json")
 json.NewEncoder(w).Encode(map[string]any{"chunk":chunk,"before_bytes":before,"after_bytes":after,"before_count":beforeCount,"after_count":afterCount,"before_count_ms":beforeMS,"after_count_ms":afterMS,"duration_ms":float64(time.Since(started).Microseconds())/1000})
}

func (h *CompressionExecuteHandler) inferenceRunRetention(w http.ResponseWriter,r *http.Request,tenant string,cutoff int64,confirm string) {
 if tenant=="" || cutoff<=0 || cutoff>time.Now().Add(-30*24*time.Hour).UnixMilli() || confirm!="DELETE INFERENCE RUNS "+tenant {http.Error(w,"tenant, cutoff older than 30 days and explicit confirmation required",400);return}
 ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
 var deleted int64
 started:=time.Now()
 err:=h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  var ids []uint
  if err:=tx.Model(&model.IntelligenceInferenceRun{}).Where("tenant_id = ? AND created_at < ?",tenant,time.UnixMilli(cutoff)).Order("created_at,id").Limit(500).Pluck("id",&ids).Error;err!=nil{return err}
  if len(ids)==0{return nil}
  res:=tx.Where("tenant_id = ? AND created_at < ? AND id IN ?",tenant,time.UnixMilli(cutoff),ids).Delete(&model.IntelligenceInferenceRun{})
  deleted=res.RowsAffected
  return res.Error
 })
 if err!=nil{http.Error(w,"inference run retention failed",500);return}
 w.Header().Set("Content-Type","application/json")
 json.NewEncoder(w).Encode(map[string]any{"updated":deleted,"duration_ms":float64(time.Since(started).Microseconds())/1000})
}

func (h *CompressionExecuteHandler) rowstore(w http.ResponseWriter,r *http.Request,confirm string) {
 if confirm!="DECOMPRESS GLOBAL EVENTS" {http.Error(w,"global confirmation required",400);return}
 if h.DB.Dialector.Name()!="postgres" {http.Error(w,"TimescaleDB required",400);return}
 ctx,cancel:=context.WithTimeout(r.Context(),120*time.Second);defer cancel()
 var chunk string
 var before,after,beforeCount,afterCount int64
 var beforeMS,afterMS float64
 started:=time.Now()
 err:=h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  var acquired bool
  if err:=tx.Raw("SELECT pg_try_advisory_xact_lock(hashtextextended('eye:events:columnstore',0))").Scan(&acquired).Error;err!=nil{return err}
  if !acquired{return gorm.ErrInvalidTransaction}
  if err:=tx.Exec("SET LOCAL lock_timeout = '3s'").Error;err!=nil{return err}
  if err:=tx.Exec("SET LOCAL statement_timeout = '110s'").Error;err!=nil{return err}
  if err:=tx.Raw("SELECT format('%I.%I',chunk_schema,chunk_name) FROM timescaledb_information.chunks WHERE hypertable_schema=current_schema() AND hypertable_name='events' AND is_compressed ORDER BY range_end DESC LIMIT 1").Scan(&chunk).Error;err!=nil{return err}
  if chunk==""{return nil}
  if err:=tx.Raw("SELECT hypertable_size('events')").Scan(&before).Error;err!=nil{return err}
  t:=time.Now()
  if err:=tx.Raw("SELECT COUNT(*) FROM "+chunk).Scan(&beforeCount).Error;err!=nil{return err}
  beforeMS=float64(time.Since(t).Microseconds())/1000
  if err:=tx.Exec("CALL convert_to_rowstore(CAST(? AS regclass))",chunk).Error;err!=nil{return err}
  t=time.Now()
  if err:=tx.Raw("SELECT COUNT(*) FROM "+chunk).Scan(&afterCount).Error;err!=nil{return err}
  afterMS=float64(time.Since(t).Microseconds())/1000
  if beforeCount!=afterCount{return gorm.ErrInvalidData}
  return tx.Raw("SELECT hypertable_size('events')").Scan(&after).Error
 })
 if err!=nil{log.Printf("rowstore conversion: %v",err);http.Error(w,"rowstore conversion failed",500);return}
 w.Header().Set("Content-Type","application/json")
 json.NewEncoder(w).Encode(map[string]any{"chunk":chunk,"before_bytes":before,"after_bytes":after,"before_count":beforeCount,"after_count":afterCount,"before_count_ms":beforeMS,"after_count_ms":afterMS,"duration_ms":float64(time.Since(started).Microseconds())/1000})
}
