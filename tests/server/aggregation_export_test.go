package test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/service"
)

type aggregationExportClient struct {
	URL     string
	Payload map[string]any
}

func (c *aggregationExportClient) PostJSON(url string, payload map[string]any) error {
	c.URL, c.Payload = url, payload
	return nil
}

func TestExportAggregateCountsIsTenantScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&model.AggregateCount{})
	db.Create(&model.AggregateCount{TenantID: "t1", EventName: "click", DimsKey: "web", Count: 5})
	db.Create(&model.AggregateCount{TenantID: "t2", EventName: "click", DimsKey: "secret", Count: 99})
	client := &aggregationExportClient{}

	err = (&service.AggregatorService{DB: db}).ExportAggregateCounts("t1", "click", "https://hooks.example/export", client)
	if err != nil {
		t.Fatal(err)
	}
	if client.URL != "https://hooks.example/export" || len(client.Payload["aggregates"].([]model.AggregateCount)) != 1 {
		t.Fatalf("unexpected export payload: %+v", client.Payload)
	}
}
