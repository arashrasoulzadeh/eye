package service

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

type IntelligenceTagRef struct {
	TenantID string `json:"tenant_id"`
	ID       string `json:"id"`
}

// IntelligenceTags returns results in request order. User references identify
// a session so the lookup uses the same tenant and identity as local scoring.
func (s *UserIntelligenceService) IntelligenceTags(ctx context.Context, kind string, refs []IntelligenceTagRef) ([][]string, error) {
	db := s.DB.WithContext(ctx)
	identities := make(map[IntelligenceTagRef]string, len(refs))
	pairs := make([]any, 0, len(refs)*2)
	for _, ref := range refs {
		identities[ref] = ref.ID
		pairs = append(pairs, ref.TenantID, ref.ID)
	}
	tags := make([][]string, len(refs))
	if len(pairs) == 0 {
		return tags, nil
	}
	if kind == "user" {
		var sessions []model.Session
		if err := db.Select(`id, tenant_id, ip, user_agent, COALESCE(NULLIF(user_id, ''),
			(SELECT e.user_id FROM events e WHERE e.tenant_id = sessions.tenant_id
			AND e.session_id = sessions.id AND e.user_id <> '' AND e.is_aggregated = ?
			ORDER BY e.client_ts, e.event_id LIMIT 1), '') AS user_id`, false).
			Where("(tenant_id, id) IN (VALUES "+tagPlaceholders(len(pairs)/2)+")", pairs...).Find(&sessions).Error; err != nil {
			return nil, err
		}
		clear(identities)
		pairs = pairs[:0]
		for _, session := range sessions {
			id := session.UserID
			if id == "" && (session.IP != "" || session.UserAgent != "") {
				id = UserFingerprint(session.IP, session.UserAgent)
			}
			if id == "" {
				continue
			}
			identities[IntelligenceTagRef{TenantID: session.TenantID, ID: session.ID}] = id
			pairs = append(pairs, session.TenantID, id)
		}
	}
	if len(pairs) == 0 {
		return tags, nil
	}
	var rows []model.IntelligenceInference
	if err := db.Select("tenant_id", "entity_id", "tags").Where("kind = ?", kind).Where("(tenant_id, entity_id) IN (VALUES "+tagPlaceholders(len(pairs)/2)+")", pairs...).Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := make(map[IntelligenceTagRef][]string, len(rows))
	for _, row := range rows {
		var values []string
		if json.Unmarshal([]byte(row.Tags), &values) == nil {
			slices.Sort(values)
			byID[IntelligenceTagRef{TenantID: row.TenantID, ID: row.EntityID}] = slices.Compact(values)
		}
	}
	for i, ref := range refs {
		tags[i] = byID[IntelligenceTagRef{TenantID: ref.TenantID, ID: identities[ref]}]
	}
	return tags, nil
}

func tagPlaceholders(count int) string {
	return strings.TrimSuffix(strings.Repeat("(?,?),", count), ",")
}
