package handler

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "testing"
 "time"
 "github.com/arashrasoulzadeh/eye/internal/model"
 "gorm.io/driver/sqlite"
 "gorm.io/gorm"
)

func TestInferenceRunRetentionIsolation(t *testing.T) {
 db,err:=gorm.Open(sqlite.Open(":memory:"),&gorm.Config{})
 if err!=nil{t.Fatal(err)}
 sqlDB,err:=db.DB();if err!=nil{t.Fatal(err)};defer sqlDB.Close();sqlDB.SetMaxOpenConns(1)
 if err=db.AutoMigrate(&model.IntelligenceInferenceRun{},&model.IntelligenceInference{});err!=nil{t.Fatal(err)}
 now:=time.Now()
 for _,row:=range []model.IntelligenceInferenceRun{{TenantID:"one",CreatedAt:now.Add(-60*24*time.Hour)},{TenantID:"two",CreatedAt:now.Add(-60*24*time.Hour)},{TenantID:"one",CreatedAt:now}} {
  if err=db.Create(&row).Error;err!=nil{t.Fatal(err)}
 }
 if err=db.Create(&model.IntelligenceInference{TenantID:"one",Kind:"event",EntityID:"a",ScoredAt:now.Add(-60*24*time.Hour)}).Error;err!=nil{t.Fatal(err)}
 h:=&CompressionExecuteHandler{DB:db}
 request:=func(action,confirm string,cutoff int64)int{
  body,_:=json.Marshal(map[string]any{"action":action,"tenant_id":"one","confirm":confirm,"to_ts":cutoff})
  w:=httptest.NewRecorder()
  h.ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/compression/prune",bytes.NewReader(body)))
  return w.Code
 }
 cutoff:=now.Add(-31*24*time.Hour).UnixMilli()
 if got:=request("inference_run_retention","wrong",cutoff);got!=400{t.Fatalf("wrong confirmation: %d",got)}
 if got:=request("inference_run_retention","DELETE INFERENCE RUNS one",now.UnixMilli());got!=400{t.Fatalf("unsafe cutoff: %d",got)}
 if got:=request("inference_run_retention","DELETE INFERENCE RUNS one",cutoff);got!=200{t.Fatalf("retention: %d",got)}
 if got:=request("inference_run_retention","DELETE INFERENCE RUNS one",cutoff);got!=200{t.Fatalf("repeat: %d",got)}
 if got:=request("columnstore","COMPRESS GLOBAL EVENTS",cutoff);got!=400{t.Fatalf("sqlite compression: %d",got)}
 var count int64
 if err=db.Model(&model.IntelligenceInferenceRun{}).Count(&count).Error;err!=nil||count!=2{t.Fatalf("remaining runs %d: %v",count,err)}
 if err=db.Model(&model.IntelligenceInference{}).Count(&count).Error;err!=nil||count!=1{t.Fatalf("current inference %d: %v",count,err)}
}
