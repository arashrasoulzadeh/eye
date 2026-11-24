package service

import (
 "context"
 "fmt"
 "time"

 "github.com/arashrasoulzadeh/eye/internal/model"
 "gorm.io/gorm"
 "gorm.io/gorm/clause"
)

// CompactEventBatch preserves count rollups, not individual event details.
// Only complete historical hours are eligible. Callers must explicitly
// authorize loss of payloads, dimensions, and session-level history.
func CompactEventBatch(ctx context.Context, db *gorm.DB, tenant, event string, cutoff int64, limit int) (int64,error) {
 if tenant == "" || event == "" || cutoff <= 0 || cutoff % 3600000 != 0 || cutoff > time.Now().Add(-24*time.Hour).UnixMilli() || limit < 1 || limit > 500 {
  return 0, fmt.Errorf("invalid compaction scope")
 }
 var removed int64
 err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  q := tx.Where("tenant_id = ? AND event_name = ? AND client_ts < ? AND is_aggregated = ?",tenant,event,cutoff,false).Order("client_ts, event_id").Limit(limit)
  if tx.Dialector.Name() == "postgres" { q = q.Clauses(clause.Locking{Strength:"UPDATE"}) }
  var rows []model.Event
  if err := q.Find(&rows).Error; err != nil { return err }
  for _, row := range rows {
   for _, spec := range []struct{name string; width int64}{{"minute",60000},{"10minute",600000},{"hour",3600000}} {
    bucket := model.CompactedEventBucket(model.EventBucket{TenantID:tenant,EventName:event,Bucket:spec.name,BucketStart:row.ClientTS-row.ClientTS%spec.width,Count:1,UpdatedAt:time.Now()})
    if err := tx.Clauses(clause.OnConflict{Columns:[]clause.Column{{Name:"tenant_id"},{Name:"event_name"},{Name:"bucket"},{Name:"bucket_start"}},DoUpdates:clause.Assignments(map[string]any{"count":gorm.Expr("compacted_event_buckets.count + excluded.count"),"updated_at":gorm.Expr("excluded.updated_at")})}).Create(&bucket).Error; err != nil { return err }
   }
   result := tx.Where("tenant_id = ? AND event_name = ? AND event_id = ? AND ingest_ts = ? AND client_ts = ? AND is_aggregated = ?",tenant,event,row.EventID,row.IngestTS,row.ClientTS,false).Delete(&model.Event{})
   if result.Error != nil { return result.Error }
   if result.RowsAffected != 1 { return fmt.Errorf("compaction record changed") }
   removed++
  }
  return nil
 })
 if err != nil { return 0,err }
 return removed,nil
}
