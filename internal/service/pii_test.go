package service

import "testing"

func TestRedactPIIRecursesThroughObjectsAndArrays(t *testing.T) {
	input := map[string]any{"email": "a@example.com", "profile": map[string]any{"phone": "123", "city": "Tehran"}, "items": []any{map[string]any{"password": "secret"}}}
	got := RedactPII(input, []string{"email", "phone", "password"}).(map[string]any)
	if got["email"] != "[REDACTED]" {
		t.Fatalf("email was not redacted: %+v", got)
	}
	profile := got["profile"].(map[string]any)
	if profile["phone"] != "[REDACTED]" || profile["city"] != "Tehran" {
		t.Fatalf("nested fields wrong: %+v", profile)
	}
	items := got["items"].([]any)[0].(map[string]any)
	if items["password"] != "[REDACTED]" {
		t.Fatalf("array field was not redacted: %+v", items)
	}
}

func TestDetectPIIFieldsNested(t *testing.T) {
	fields := DetectPIIFields(map[string]any{"profile": map[string]any{"emailAddress": "a@example.com"}, "items": []any{map[string]any{"access_token": "secret"}}, "city": "Tehran"})
	got := map[string]bool{}
	for _, field := range fields {
		got[field] = true
	}
	if !got["emailAddress"] || !got["access_token"] || len(got) != 2 {
		t.Fatalf("detected fields = %#v", fields)
	}
}
