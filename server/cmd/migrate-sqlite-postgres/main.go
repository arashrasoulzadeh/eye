package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/arashrasoulzadeh/eye/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const batchSize = 500

func main() {
	sourcePath := flag.String("sqlite", "/data/eye.db", "source SQLite database")
	databaseURL := flag.String("postgres", os.Getenv("DATABASE_URL"), "destination PostgreSQL URL")
	flag.Parse()
	if *databaseURL == "" {
		log.Fatal("-postgres or DATABASE_URL is required")
	}
	gormLogger := logger.New(log.New(io.Discard, "", 0), logger.Config{LogLevel: logger.Silent})
	source, err := gorm.Open(sqlite.Open(*sourcePath+"?_busy_timeout=10000"), &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Fatal(err)
	}
	destination, err := gorm.Open(postgres.Open(*databaseURL), &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Fatal(err)
	}
	if err := repository.New(destination).Migrate(); err != nil {
		log.Fatal(err)
	}
	models := []any{
		&model.Tenant{}, &model.Session{}, &model.Event{}, &model.Schema{}, &model.Aggregation{}, &model.Experiment{}, &model.ExperimentAssignment{}, &model.Cohort{}, &model.CohortMembership{}, &model.AggregateCount{}, &model.AlertRule{}, &model.RuntimeSettings{}, &model.WebhookDeadLetter{}, &model.AuditLog{}, &model.AdminUser{}, &model.FeatureFlag{}, &model.DashboardCache{}, &model.Question{}, &model.PendingStreamEvent{}, &model.EventBucket{}, &model.UserIntelligence{}, &model.UserIntelligenceLabel{}, &model.IntelligenceTrainingRun{}, &model.IntelligencePipelineRun{}, &model.IntelligenceModel{}, &model.IntelligenceEvaluation{}, &model.IntelligenceDataQuality{}, &model.IntelligenceAlert{}, &model.AdminTeam{}, &model.AdminTeamMember{}, &model.AdminSession{},
	}
	for _, value := range models {
		statement := destination.Session(&gorm.Session{}).Model(value)
		if err := statement.Statement.Parse(value); err != nil {
			log.Fatal(err)
		}
		table := statement.Statement.Schema.Table
		if !source.Migrator().HasTable(table) {
			continue
		}
		if err := copyTable(source, destination, table); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("SQLite to PostgreSQL migration completed; SQLite source was preserved")
}

// copyTable deliberately uses database rows instead of GORM's FindInBatches:
// the source and destination schemas are compatible but the migration must
// preserve every column (including columns added after this binary was built).
func copyTable(source, destination *gorm.DB, table string) error {
	rows, err := source.Table(table).Rows()
	if err != nil {
		return fmt.Errorf("read %s: %w", table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("columns %s: %w", table, err)
	}
	for batch := 1; ; batch++ {
		records := make([]map[string]any, 0, batchSize)
		for len(records) < batchSize && rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				return fmt.Errorf("scan %s: %w", table, err)
			}
			record := make(map[string]any, len(columns))
			for i, column := range columns {
				// SQLite drivers may return []byte backed by a reused buffer.
				if bytes, ok := values[i].([]byte); ok {
					record[column] = append([]byte(nil), bytes...)
				} else {
					record[column] = values[i]
				}
			}
			records = append(records, record)
		}
		if len(records) == 0 {
			break
		}
		if err := destination.Table(table).Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error; err != nil {
			return fmt.Errorf("copy %s batch %d: %w", table, batch, err)
		}
		log.Printf("migrated %s batch %d (%d rows)", table, batch, len(records))
	}
	return rows.Err()
}
