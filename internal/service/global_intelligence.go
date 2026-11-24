package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
 "sync"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sessionObservation = 30 * time.Minute
const returnObservation = 7 * 24 * time.Hour

type GlobalIntelligenceBundle struct {
	SequenceNext map[string]string  `json:"sequence_next"`
	Schema       int                `json:"schema"`
	Version      string             `json:"version"`
	Next         map[string]string  `json:"next"`
	Conversion   map[string]float64 `json:"conversion"`
	Return       map[string]float64 `json:"return"`
}

var trainingAliases = map[string]string{
	"page_view": "navigation", "session_start": "navigation", "click": "engagement",
	"scroll_depth": "engagement", "product_view": "product_impression",
	"view_product": "product_impression", "add_to_cart": "cart_add",
	"purchase": "purchase_confirmed", "order_completed": "purchase_confirmed",
	"begin_checkout": "checkout_started", "js_error": "error",
	"search": "search_query", "combo_milestone": "combo_reached",
}

func trainingEvent(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if category, ok := trainingAliases[name]; ok {
		return category
	}
	if globalEventTags[name] {
		return name
	}
	return "other"
}

type trainingJourney struct {
	Session   model.Session
	Events    []model.Event
	Sequence  []string
	Gaps      []string
	Purchased bool
}

func (s *UserIntelligenceService) trainingJourneys(db *gorm.DB, tenant string, visit func(trainingJourney) error) error {
	return s.trainingJourneysScoped(db,tenant,nil,visit)
}

func (s *UserIntelligenceService) trainingJourneysScoped(db *gorm.DB, tenant string, sessionIDs []string, visit func(trainingJourney) error) error {
 var sessions []model.Session
 query:=db.Where("tenant_id = ?",tenant)
 if sessionIDs!=nil {
  if len(sessionIDs)==0{return nil}
  query=query.Where("id IN ?",sessionIDs)
 }
 return query.FindInBatches(&sessions, 200, func(_ *gorm.DB, _ int) error {
		ids := make([]string, 0, len(sessions))
		for _, session := range sessions {
			ids = append(ids, session.ID)
		}
		var events []model.Event
		if err := db.Select("event_id", "session_id", "user_id", "event_name", "client_ts").
			Where("tenant_id = ? AND session_id IN ? AND is_aggregated = ?", tenant, ids, false).
			Order("client_ts, event_id").Find(&events).Error; err != nil {
			return err
		}
		grouped := make(map[string][]model.Event)
		for _, event := range events {
			grouped[event.SessionID] = append(grouped[event.SessionID], event)
		}
		for _, session := range sessions {
			journey := trainingJourney{Session: session, Events: grouped[session.ID]}
			for i, event := range journey.Events {
				category := trainingEvent(event.EventName)
				journey.Sequence = append(journey.Sequence, category)
				journey.Purchased = journey.Purchased || category == "purchase_confirmed"
				if journey.Session.UserID == "" && event.UserID != "" {
					journey.Session.UserID = event.UserID
				}
				if i > 0 {
					gap := event.ClientTS - journey.Events[i-1].ClientTS
					switch {
					case gap < 1000:
						journey.Gaps = append(journey.Gaps, "0")
					case gap < 10000:
						journey.Gaps = append(journey.Gaps, "1")
					case gap < 60000:
						journey.Gaps = append(journey.Gaps, "2")
					default:
						journey.Gaps = append(journey.Gaps, "3")
					}
				}
			}
			if err := visit(journey); err != nil {
				return err
			}
		}
		return nil
	}).Error
}

func scopedTenants(db *gorm.DB, tenant string) ([]string, error) {
	if tenant != "" {
		return []string{tenant}, nil
	}
	var ids []string
	err := db.Model(&model.Tenant{}).Order("id").Pluck("id", &ids).Error
	return ids, err
}

func (s *UserIntelligenceService) ExportTrainingCSV(out io.Writer, tenant string, now time.Time, bounds ...int64) error {
	ids, err := scopedTenants(s.DB, tenant)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(out)
	if err := writer.Write([]string{"schema_version", "event_sequence", "time_gap_buckets", "purchased", "returned_within_7d", "prior_visits"}); err != nil {
		return err
	}
	for _, id := range ids {
		var visits []model.Session
		if err := s.DB.Select("id", "user_id", "started_at").Where("tenant_id = ? AND user_id <> ?", id, "").Order("started_at, id").Find(&visits).Error; err != nil {
			return err
		}
		history := map[string][]model.Session{}
		positions := map[string]int{}
		for _, visit := range visits {
			positions[visit.ID] = len(history[visit.UserID])
			history[visit.UserID] = append(history[visit.UserID], visit)
		}
		err := s.trainingJourneys(s.DB, id, func(journey trainingJourney) error {
			if len(bounds) == 2 && (journey.Session.StartedAt.UnixMilli() < bounds[0] || journey.Session.LastSeen.UnixMilli() > bounds[1]) {
				return nil
			}
			if len(journey.Sequence) == 0 {
				return nil
			}
			purchased, returned, prior := "", "", ""
			if journey.Purchased {
				purchased = "1"
			} else if !journey.Session.LastSeen.IsZero() && now.Sub(journey.Session.LastSeen) >= sessionObservation {
				purchased = "0"
			}
			if index, ok := positions[journey.Session.ID]; ok {
				prior = strconv.Itoa(min(index, 5))
				userVisits := history[journey.Session.UserID]
				if index+1 < len(userVisits) && userVisits[index+1].StartedAt.Sub(journey.Session.StartedAt) <= returnObservation {
					returned = "1"
				} else if now.Sub(journey.Session.StartedAt) >= returnObservation {
					returned = "0"
				}
			}
			return writer.Write([]string{"1", strings.Join(journey.Sequence, "|"), strings.Join(journey.Gaps, "|"), purchased, returned, prior})
		})
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func inferenceJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

var globalScoringMu sync.Mutex

func (s *UserIntelligenceService) ScoreGlobalModel(ctx context.Context, bundle GlobalIntelligenceBundle, now time.Time) error {
 if !globalScoringMu.TryLock() { return errors.New("global intelligence scoring already running") }; defer globalScoringMu.Unlock()
	db := s.DB.WithContext(ctx)
	ids, err := scopedTenants(db, "")
	if err != nil {
		return err
	}
	var failures []error
	for _, tenant := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		run := model.IntelligenceInferenceRun{TenantID: tenant, Version: bundle.Version, CreatedAt: now}
        skipped := false
		err := db.Transaction(func(tx *gorm.DB) error {
			if tx.Dialector.Name()=="postgres" {
                var acquired bool
                if err:=tx.Raw("SELECT pg_try_advisory_xact_lock(hashtextextended(?, 0))", "eye:global-scoring:"+tenant).Scan(&acquired).Error;err!=nil{return err}
                if !acquired{return errors.New("tenant scoring already running on another instance")}
            }
			if tx.Dialector.Name()=="postgres" {
                // Bound redundant full scans; always reconcile after the interval.
                // This is throttling, not an event-level incremental checkpoint.
                var recent int64
                if err:=tx.Model(&model.IntelligenceInferenceRun{}).Where("tenant_id = ? AND version = ? AND error = ? AND created_at > ? AND created_at <= ?",tenant,bundle.Version,"",now.Add(-time.Minute),now).Count(&recent).Error;err!=nil{return err}
                if recent>0 {skipped=true;return nil}
                if err:=tx.Exec("CREATE TEMP TABLE eye_scored_entities (kind text NOT NULL, entity_id text NOT NULL, PRIMARY KEY(kind,entity_id)) ON COMMIT DROP").Error;err!=nil{return err}
            }
			if err := s.scoreGlobalTenant(tx, bundle, &run, now); err != nil {
				return err
			}
			if tx.Dialector.Name()=="postgres" {
                return tx.Exec("DELETE FROM intelligence_inferences i WHERE i.tenant_id = ? AND NOT EXISTS (SELECT 1 FROM pg_temp.eye_scored_entities k WHERE k.kind=i.kind AND k.entity_id=i.entity_id)",tenant).Error
            }
            return tx.Where("tenant_id = ? AND scored_at < ?", tenant, now).Delete(&model.IntelligenceInference{}).Error
		})
        if skipped {continue}
		if err != nil {
			run.Error = err.Error()
			run.Events, run.Sessions, run.Users = 0, 0, 0
		}
		if logErr := db.Create(&run).Error; logErr != nil {
			return logErr
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("score tenant %s: %w", tenant, err))
		}
	}
	return errors.Join(failures...)
}

func (s *UserIntelligenceService) scoreGlobalTenant(db *gorm.DB, bundle GlobalIntelligenceBundle, run *model.IntelligenceInferenceRun, now time.Time) error {
	type profile struct{ visits, purchases, games, adSessions int }
	users := map[string]profile{}
	batchSize:=500
	if db.Dialector.Name()=="sqlite" {batchSize=100}
	results := make([]model.IntelligenceInference, 0, batchSize)
	flush := func() error {
		if len(results) == 0 {
			return nil
		}
		// Batch inference writes and explicitly preserve the composite identity.
		// PostgreSQL tracks membership separately so unchanged snapshots need no update.
		conflict := clause.OnConflict{
			Columns: []clause.Column{{Name:"tenant_id"},{Name:"kind"},{Name:"entity_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"version","tags","predictions","evidence","scored_at"}),
		 }
        if db.Dialector.Name()=="postgres" {
            type scoredKey struct { Kind string; EntityID string }
            keys:=make([]scoredKey,0,len(results))
            for _,row:=range results {keys=append(keys,scoredKey{row.Kind,row.EntityID})}
            if err:=db.Table("pg_temp.eye_scored_entities").Clauses(clause.OnConflict{DoNothing:true}).Create(&keys).Error;err!=nil{return err}
            // Compare the batch against persisted snapshots before attempting writes.
            // Keep membership for every entity, including unchanged snapshots.
            var existing []model.IntelligenceInference
            ids:=make([]string,0,len(results))
            for _,row:=range results {ids=append(ids,row.EntityID)}
            if err:=db.Select("kind","entity_id","version","tags","predictions","evidence").Where("tenant_id = ? AND entity_id IN ?",run.TenantID,ids).Find(&existing).Error;err!=nil{return err}
            type identity struct {kind,id string}
            previous:=make(map[identity]model.IntelligenceInference,len(existing))
            for _,row:=range existing {previous[identity{row.Kind,row.EntityID}]=row}
            changed:=results[:0]
            for _,row:=range results {
                old,ok:=previous[identity{row.Kind,row.EntityID}]
                if !ok || old.Version!=row.Version || old.Tags!=row.Tags || old.Predictions!=row.Predictions || old.Evidence!=row.Evidence {changed=append(changed,row)}
            }
            results=changed
            if len(results)==0 {return nil}
            conflict.Where=clause.Where{Exprs:[]clause.Expression{clause.Expr{SQL:"(intelligence_inferences.version, intelligence_inferences.tags, intelligence_inferences.predictions, intelligence_inferences.evidence) IS DISTINCT FROM (excluded.version, excluded.tags, excluded.predictions, excluded.evidence)"}}}
        }
        err:=db.Clauses(conflict).Create(&results).Error
        results = results[:0]
		return err
	}
	add := func(kind, id string, tags []string, predictions, evidence map[string]any) error {
		results = append(results, model.IntelligenceInference{TenantID: run.TenantID, Kind: kind, EntityID: id, Version: bundle.Version, Tags: inferenceJSON(tags), Predictions: inferenceJSON(predictions), Evidence: inferenceJSON(evidence), ScoredAt: now})
		if len(results) == cap(results) {
			return flush()
		}
		return nil
	}
	err := s.trainingJourneys(db, run.TenantID, func(journey trainingJourney) error {
		tags := []string{}
		seen := map[string]bool{}
		for i, category := range journey.Sequence {
			seen[category] = true
			eventTags := []string{}
			if category != "other" {
				eventTags = append(eventTags, category)
			}
			if i == 0 && category != "entry_point" {
				eventTags = append(eventTags, "entry_point")
			}
			if err := add("event", journey.Events[i].EventID, eventTags, nil, map[string]any{"category": category}); err != nil {
				return err
			}
			run.Events++
		}
		if seen["navigation"] || seen["product_impression"] {
			tags = append(tags, "browsing")
		}
		if seen["game_started"] || seen["question_answered"] || seen["match_found"] || seen["precise_hit"] {
			tags = append(tags, "gameplay")
		}
		if journey.Purchased {
			tags = append(tags, "converted")
		}
		if seen["checkout_started"] {
			tags = append(tags, "checkout")
		}
		if seen["error"] || seen["client_crash"] {
			tags = append(tags, "error_encountered")
		}
		if seen["combo_reached"] {
			tags = append(tags, "high_combo_session")
		}
		for _, rule := range sessionEventTags {
			if seen[rule.event] {
				tags = append(tags, rule.tag)
			}
		}
		if seen["cart_add"] && !journey.Purchased && !journey.Session.LastSeen.IsZero() && now.Sub(journey.Session.LastSeen) >= sessionObservation {
			tags = append(tags, "cart_abandoned")
		}
		if seen["matchmaking_queued"] && !seen["match_found"] && now.Sub(journey.Session.LastSeen) >= sessionObservation {
			tags = append(tags, "matchmaking_abandoned")
		}
		identity := journey.Session.UserID
		if identity == "" && (journey.Session.IP != "" || journey.Session.UserAgent != "") {
			identity = UserFingerprint(journey.Session.IP, journey.Session.UserAgent)
		}
		if identity != "" {
			profile := users[identity]
			profile.visits++
			if journey.Purchased {
				profile.purchases++
			}
			if seen["game_started"] || seen["question_answered"] || seen["match_found"] {
				profile.games++
			}
			if seen["rewarded_ad_completed"] {
				profile.adSessions++
			}
			users[identity] = profile
		}
		predictions := map[string]any{}
		if len(journey.Sequence) > 0 && now.Sub(journey.Session.LastSeen) < sessionObservation {
			last := journey.Sequence[len(journey.Sequence)-1]
			next := bundle.SequenceNext[strings.Join(journey.Sequence[max(0, len(journey.Sequence)-3):], "|")]
			if next == "" {
				next = bundle.Next[last]
			}
			if next != "" {
				predictions["next_event"] = next
			}
			if p, ok := bundle.Conversion[last]; ok && !journey.Purchased {
				predictions["session_conversion"] = p
			}
		}
		run.Sessions++
		return add("session", journey.Session.ID, tags, predictions, map[string]any{"events": len(journey.Sequence)})
	})
	if err != nil {
		return err
	}
	for id, profile := range users {
		tags := []string{}
		if profile.visits > 1 {
			tags = append(tags, "returning")
		} else {
			tags = append(tags, "first_time_visitor")
		}
		if profile.visits >= 5 {
			tags = append(tags, "frequent_visitor")
		}
		if profile.adSessions >= 3 {
			tags = append(tags, "rewarded_ad_regular")
		}
		if profile.purchases > 0 {
			tags = append(tags, "buyer")
		}
		if profile.purchases > 1 {
			tags = append(tags, "repeat_buyer")
		}
		if profile.games > 1 {
			tags = append(tags, "returning_player")
		} else if profile.games == 1 {
			tags = append(tags, "new_player")
		}
		predictions := map[string]any{}
		if p, ok := bundle.Return[strconv.Itoa(min(profile.visits-1, 5))]; ok {
			predictions["return_within_7d"] = p
		}
		if err := add("user", id, tags, predictions, map[string]any{"sessions": profile.visits, "purchase_sessions": profile.purchases, "game_sessions": profile.games}); err != nil {
			return err
		}
		run.Users++
	}
	return flush()
}
