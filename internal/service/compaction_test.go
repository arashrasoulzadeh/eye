package service

import (
 "context"
 "testing"
 "time"

 "github.com/arashrasoulzadeh/eye/internal/model"
 "gorm.io/driver/sqlite"
 "gorm.io/gorm"
)

func TestCompactEventBatchPreservesHistoricalCounts(t *testing.T) {
 db,err:=gorm.Open(sqlite.Open(":memory:"),&gorm.Config{})
 if err!=nil { t.Fatal(err) }
 sqlDB,err:=db.DB(); if err!=nil {t.Fatal(err)}; defer sqlDB.Close()
 sqlDB.SetMaxOpenConns(1)
 if err=db.AutoMigrate(&model.Event{},&model.EventBucket{},&model.CompactedEventBucket{});err!=nil {t.Fatal(err)}
 cutoff:=time.Now().Add(-48*time.Hour).Truncate(time.Hour).UnixMilli()
 for _,e:=range []model.Event{
 {EventID:"a",TenantID:"one",EventName:"search",ClientTS:cutoff-60000,IngestTS:time.Now()},
 {EventID:"b",TenantID:"two",EventName:"search",ClientTS:cutoff-60000,IngestTS:time.Now()},
 } { if err=db.Create(&e).Error;err!=nil {t.Fatal(err)} }
 svc:=&BucketService{DB:db}
 if err=svc.Backfill();err!=nil {t.Fatal(err)}
 n,err:=CompactEventBatch(context.Background(),db,"one","search",cutoff,500)
 if err!=nil || n!=1 {t.Fatalf("removed=%d err=%v",n,err)}
 n,err=CompactEventBatch(context.Background(),db,"one","search",cutoff,500)
 if err!=nil || n!=0 {t.Fatalf("repeat removed=%d err=%v",n,err)}
 if err=svc.Backfill();err!=nil {t.Fatal(err)}
 var count int64
 if err=db.Model(&model.Event{}).Where("tenant_id = ?","two").Count(&count).Error;err!=nil || count!=1 {t.Fatalf("other tenant count=%d err=%v",count,err)}
 var bucket model.EventBucket
 if err=db.Where("tenant_id = ? AND event_name = ? AND bucket = ?","one","search","minute").First(&bucket).Error;err!=nil || bucket.Count!=1 {t.Fatalf("restored bucket=%+v err=%v",bucket,err)}
}

func TestCompactEventBatchRollback(t *testing.T) {
 db,err:=gorm.Open(sqlite.Open(":memory:"),&gorm.Config{})
 if err!=nil {t.Fatal(err)}
 sqlDB,err:=db.DB(); if err!=nil {t.Fatal(err)}; defer sqlDB.Close()
 sqlDB.SetMaxOpenConns(1)
 if err=db.AutoMigrate(&model.Event{},&model.CompactedEventBucket{});err!=nil {t.Fatal(err)}
 cutoff:=time.Now().Add(-48*time.Hour).Truncate(time.Hour).UnixMilli()
 e:=model.Event{EventID:"rollback",TenantID:"one",EventName:"search",ClientTS:cutoff-60000,IngestTS:time.Now()}
 if err=db.Create(&e).Error;err!=nil {t.Fatal(err)}
 if err=db.Exec("CREATE TRIGGER reject_event_delete BEFORE DELETE ON events BEGIN SELECT RAISE(ABORT, 'forced failure'); END").Error;err!=nil {t.Fatal(err)}
 n,err:=CompactEventBatch(context.Background(),db,"one","search",cutoff,500)
 if err==nil || n!=0 {t.Fatalf("expected rollback, removed=%d err=%v",n,err)}
 var count int64
 if err=db.Model(&model.Event{}).Count(&count).Error;err!=nil || count!=1 {t.Fatalf("raw count=%d err=%v",count,err)}
 if err=db.Model(&model.CompactedEventBucket{}).Count(&count).Error;err!=nil || count!=0 {t.Fatalf("rollup count=%d err=%v",count,err)}
}
