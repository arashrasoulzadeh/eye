package service

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/eye/eye/internal/model"
)

type AggregatorService struct{ DB *gorm.DB }

func (a *AggregatorService) Create(name, tenantID, rule, output string, threshold int, window string) (*model.Aggregation, error) {
	agg := &model.Aggregation{Name: name, TenantID: tenantID, Rule: rule, OutputEvent: output, Threshold: threshold, Window: window, Enabled: true, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}
	if err := a.DB.Create(agg).Error; err != nil { return nil, err }
	return agg, nil
}

func (a *AggregatorService) List(tenantID string) ([]model.Aggregation, error) {
	var list []model.Aggregation
	if err := a.DB.Where("tenant_id = ?", tenantID).Find(&list).Error; err != nil { return nil, err }
	return list, nil
}

func (a *AggregatorService) EmitSystemEvent(tenantID, name string, payload map[string]any) error {
	b, _ := json.Marshal(payload)
	ev := &model.Event{EventID: "sys-" + time.Now().Format("20060102150405"), TenantID: tenantID, EventName: name, ClientTS: time.Now().UnixMilli(), IngestTS: time.Now(), Payload: string(b), Source: "system", IsAggregated: true}
	return a.DB.Create(ev).Error
}
