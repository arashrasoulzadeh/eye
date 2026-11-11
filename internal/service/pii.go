package service

import "strings"

// RedactPII recursively replaces configured payload keys before persistence.
func RedactPII(value any, fields []string) any {
	keys := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(strings.ToLower(field))
		if field != "" {
			keys[field] = struct{}{}
		}
	}
	return redactPII(value, keys)
}

func redactPII(value any, keys map[string]struct{}) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if _, ok := keys[strings.ToLower(key)]; ok {
				out[key] = "[REDACTED]"
			} else {
				out[key] = redactPII(item, keys)
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactPII(item, keys)
		}
		return out
	default:
		return value
	}
}
