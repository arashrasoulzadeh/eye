package service

import "strings"

var detectedPIIKeyFragments = []string{"email", "phone", "mobile", "password", "passwd", "token", "secret", "ssn", "social_security", "credit_card", "card_number"}

// DetectPIIFields returns payload keys that look sensitive by name. It is
// intentionally conservative and key-based: values are never logged or sent
// to a third-party detector. Callers can combine the result with configured
// fields and redact before persistence.
func DetectPIIFields(value any) []string {
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, item := range typed {
				lower := strings.ToLower(strings.TrimSpace(key))
				for _, fragment := range detectedPIIKeyFragments {
					if strings.Contains(lower, fragment) {
						seen[key] = struct{}{}
						break
					}
				}
				walk(item)
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(value)
	fields := make([]string, 0, len(seen))
	for field := range seen {
		fields = append(fields, field)
	}
	return fields
}

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
