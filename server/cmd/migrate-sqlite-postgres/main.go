package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const batchSize = 500

func main() {
	sourcePath := flag.String("sqlite", "/data/eye.db", "source SQLite database")
	databaseURL := flag.String("postgres", os.Getenv("DATABASE_URL"), "destination PostgreSQL URL")
	flag.Parse()
	if *databaseURL == "" {
		log.Fatal("-postgres or DATABASE_URL is required")
	}
	source, err := gorm.Open(sqlite.Open(*sourcePath+"?_busy_timeout=10000"), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	destination, err := gorm.Open(postgres.Open(*databaseURL), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	if err := repository.New(destination).Migrate(); err != nil { log.Fatal(err) }
	models := []any{
		&model.Tenant{}, &model.Session{}, &model.Event{}, &model.Schema{}, &model.Aggregation{}, &model.Experiment{}, &model.ExperimentAssignment{}, &model.Cohort{}, &model.CohortMembership{}, &model.AggregateCount{}, &model.AlertRule{}, &model.RuntimeSettings{}, &model.WebhookDeadLetter{}, &model.AuditLog{}, &model.AdminUser{}, &model.FeatureFlag{}, &model.DashboardCache{}, &model.Question{}, &model.PendingStreamEvent{}, &model.EventBucket{}, &model.UserIntelligence{}, &model.UserIntelligenceLabel{}, &model.IntelligenceTrainingRun{}, &model.IntelligencePipelineRun{}, &model.IntelligenceModel{}, &model.IntelligenceEvaluation{}, &model.IntelligenceDataQuality{}, &model.IntelligenceAlert{}, &model.AdminTeam{}, &model.AdminTeamMember{}, &model.AdminSession{},
	}
	for _, value := range models {
		statement := destination.Session(&gorm.Session{}).Model(value)
		if err := statement.Statement.Parse(value); err != nil { log.Fatal(err) }
		table := statement.Statement.Schema.Table
		if !source.Migrator().HasTable(table) { continue }
		var rows []map[string]any
		if err := source.Table(table).FindInBatches(&rows, batchSize, func(_ *gorm.DB, batch int) error {
			if len(rows) == 0 { return nil }
			if err := destination.Table(table).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
				return fmt.Errorf("copy %s batch %d: %w", table, batch, err)
			}
			log.Printf("migrated %s batch %d (%d rows)", table, batch, len(rows))
			return nil
		}).Error; err != nil { log.Fatal(err) }
	}
	log.Println("SQLite to PostgreSQL migration completed; SQLite source was preserved")
}
