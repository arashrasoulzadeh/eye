package handler

import (
 "context"
 "encoding/json"
 "net/http"
 "time"

 "gorm.io/gorm"
 "github.com/arashrasoulzadeh/eye/internal/model"
)

type CompressionExecuteHandler struct { DB *gorm.DB }

func (h *CompressionExecuteHandler) ServeHTTP(w http.ResponseWriter,r *http.Request) {
 if r.Method!=http.MethodPost {w.WriteHeader(405);return}
 var p struct { Tenant string `json:"tenant_id"`; Event string `json:"event_name"`; Cutoff int64 `json:"to_ts"`; Confirm string `json:"confirm"` }
 if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&p)!=nil || p.Tenant=="" || p.Event=="" || p.Cutoff<=0 || p.Cutoff>time.Now().Add(-24*time.Hour).UnixMilli() || p.Confirm!="PRUNE "+p.Tenant+" "+p.Event {http.Error(w,"explicit tenant, event, historical cutoff and confirmation required",400);return}
 ctx,cancel:=context.WithTimeout(r.Context(),15*time.Second);defer cancel()
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
