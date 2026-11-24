package server_test

import (
 "context"
 "testing"
 "github.com/arashrasoulzadeh/eye/internal/service"
 "github.com/arashrasoulzadeh/eye/internal/model"
 "gorm.io/driver/sqlite"
 "gorm.io/gorm"
)

func TestQueryExplorer(t *testing.T) {
 db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
 if err != nil { t.Fatal(err) }
 if err := db.AutoMigrate(&model.Event{}); err != nil { t.Fatal(err) }
 for _, e := range []model.Event{
 {EventID:"a",TenantID:"one",EventName:"answer",Payload:`{"streak":3}`,Dims:"{}"},
 {EventID:"b",TenantID:"two",EventName:"answer",Payload:`{"streak":9}`,Dims:"{}"},
 {EventID:"c",TenantID:"one",EventName:"answer",Payload:`{"streak":1}`,Dims:"{}"},
 } { if err := db.Create(&e).Error; err != nil { t.Fatal(err) } }
 q := &service.QueryService{DB:db}
 result, err := q.Explore(context.Background(), "one", "SELECT event_id, streak FROM events WHERE events.type = 'answer' AND event.streak > 1",0,0)
 if err != nil { t.Fatal(err) }
 if len(result.Rows)!=1 || result.Rows[0][0]!="a" { t.Fatalf("unexpected rows: %#v",result.Rows) }
 for _, sql := range []string{"DELETE FROM events","SELECT * FROM tenants","SELECT * FROM events; DELETE FROM events","SELECT * FROM events UNION SELECT * FROM tenants","SELECT pg_sleep(10) FROM events"} {
 if _,err := q.Explore(context.Background(),"one",sql,0,0); err == nil { t.Errorf("accepted %q",sql) }
 }
}
