package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

type ExperimentService struct{ DB *gorm.DB }

type SignificanceResult struct {
	ControlRate   float64 `json:"control_rate"`
	VariantRate   float64 `json:"variant_rate"`
	AbsoluteLift  float64 `json:"absolute_lift"`
	ZScore        float64 `json:"z_score"`
	PValue        float64 `json:"p_value"`
	Significant95 bool    `json:"significant_95"`
}

// Significance compares two conversion proportions with a pooled two-sided
// two-proportion z-test. Counts are supplied by the caller so the same
// calculation can be used with experiment metrics or exported aggregates.
func Significance(controlSuccesses, controlSamples, variantSuccesses, variantSamples int64) (SignificanceResult, error) {
	if controlSamples <= 0 || variantSamples <= 0 || controlSuccesses < 0 || variantSuccesses < 0 || controlSuccesses > controlSamples || variantSuccesses > variantSamples {
		return SignificanceResult{}, errors.New("invalid experiment counts")
	}
	controlRate := float64(controlSuccesses) / float64(controlSamples)
	variantRate := float64(variantSuccesses) / float64(variantSamples)
	pooled := float64(controlSuccesses+variantSuccesses) / float64(controlSamples+variantSamples)
	se := math.Sqrt(pooled * (1 - pooled) * (1/float64(controlSamples) + 1/float64(variantSamples)))
	z := 0.0
	if se > 0 {
		z = (variantRate - controlRate) / se
	}
	p := 1 - math.Erf(math.Abs(z)/math.Sqrt2)
	return SignificanceResult{ControlRate: controlRate, VariantRate: variantRate, AbsoluteLift: variantRate - controlRate, ZScore: z, PValue: p, Significant95: p < 0.05}, nil
}

func randomExperimentID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *ExperimentService) Create(name, tenantID, variants string, traffic int) (*model.Experiment, error) {
	exp := &model.Experiment{ID: randomExperimentID(), Name: name, TenantID: tenantID, Variants: variants, Traffic: traffic, Enabled: true, CreatedAt: time.Now().Unix()}
	if err := e.DB.Create(exp).Error; err != nil {
		return nil, err
	}
	return exp, nil
}

// bucket deterministically maps a key to [0, 100) using a stable hash, so the
// same (experimentID, sessionID) pair always lands in the same bucket.
func bucket(key string) int {
	sum := sha256.Sum256([]byte(key))
	n := binary.BigEndian.Uint64(sum[:8])
	return int(n % 100)
}

// AssignVariant deterministically and persistently assigns sessionID a variant
// of the given experiment based on its traffic split, so repeated calls for
// the same session always return the same variant (sticky assignment). The
// experiment's Traffic field is the percentage of sessions included in the
// experiment at all; included sessions are then split evenly across its
// variants. A session outside the traffic split gets "" (not in experiment).
func (e *ExperimentService) AssignVariant(experimentID, sessionID string) (string, error) {
	var existing model.ExperimentAssignment
	err := e.DB.First(&existing, "experiment_id = ? AND session_id = ?", experimentID, sessionID).Error
	if err == nil {
		return existing.Variant, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var exp model.Experiment
	if err := e.DB.First(&exp, "id = ?", experimentID).Error; err != nil {
		return "", err
	}

	variant := ""
	if bucket(experimentID+":"+sessionID+":include") < exp.Traffic {
		var names []string
		if err := json.Unmarshal([]byte(exp.Variants), &names); err != nil || len(names) == 0 {
			return "", errors.New("experiment has no valid variants")
		}
		idx := bucket(experimentID+":"+sessionID+":variant") % len(names)
		variant = names[idx]
	}

	assignment := &model.ExperimentAssignment{
		ExperimentID: experimentID,
		SessionID:    sessionID,
		Variant:      variant,
		AssignedAt:   time.Now().Unix(),
	}
	// Another concurrent call may have inserted first; either way, return the
	// now-persisted row's variant so the result stays sticky.
	if err := e.DB.Create(assignment).Error; err != nil {
		var raced model.ExperimentAssignment
		if err2 := e.DB.First(&raced, "experiment_id = ? AND session_id = ?", experimentID, sessionID).Error; err2 == nil {
			return raced.Variant, nil
		}
		return "", err
	}
	return variant, nil
}

// ActiveAssignmentForSession returns the most recently assigned experiment
// variant for a session, if any sticky assignment with a non-empty variant
// (i.e. the session was included in that experiment's traffic split) exists.
// Used by IngestService to auto-tag events without the client having to pass
// experiment/variant explicitly.
func (e *ExperimentService) ActiveAssignmentForSession(sessionID string) (experimentID, variant string, ok bool) {
	var a model.ExperimentAssignment
	err := e.DB.Where("session_id = ? AND variant <> ''", sessionID).
		Order("assigned_at DESC").First(&a).Error
	if err != nil {
		return "", "", false
	}
	return a.ExperimentID, a.Variant, true
}
