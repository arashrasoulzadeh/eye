package service

import "strings"

func eventUserID(event map[string]any) string {
	if value, ok := event["user_id"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if value, ok := event["userId"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if dims, ok := event["dims"].(map[string]any); ok {
		for _, key := range []string{"user_id", "userId"} {
			if value, ok := dims[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}
