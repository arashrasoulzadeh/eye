package service

import (
	"testing"
)

func TestMatchesPrimitiveType(t *testing.T) {
	cases := []struct {
		value    any
		typ      string
		expected bool
	}{
		{"hello", "string", true},
		{"hello", "number", false},
		{float64(42), "number", true},
		{float64(42), "integer", true},
		{float64(42), "string", false},
		{true, "boolean", true},
		{true, "string", false},
		{[]any{1, 2}, "array", true},
		{[]any{1, 2}, "string", false},
		{map[string]any{"a": 1}, "object", true},
		{map[string]any{"a": 1}, "string", false},
		{nil, "string", false},
		{nil, "number", false},
		{customType{}, "unknown", true},
	}

	for _, tc := range cases {
		got := matchesPrimitiveType(tc.value, tc.typ)
		if got != tc.expected {
			t.Errorf("matchesPrimitiveType(%v, %q) = %v, want %v", tc.value, tc.typ, got, tc.expected)
		}
	}
}

type customType struct{}

func TestGoType(t *testing.T) {
	cases := []struct {
		value    any
		expected string
	}{
		{"hello", "string"},
		{float64(42), "number"},
		{true, "boolean"},
		{[]any{1, 2}, "array"},
		{map[string]any{"a": 1}, "object"},
		{nil, "null"},
		{customType{}, "unknown"},
	}

	for _, tc := range cases {
		got := goType(tc.value)
		if got != tc.expected {
			t.Errorf("goType(%v) = %q, want %q", tc.value, got, tc.expected)
		}
	}
}

func TestFieldPath(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"", ""},
		{"a", "a: "},
		{"a.b", "a.b: "},
		{"a.b.c", "a.b.c: "},
	}

	for _, tc := range cases {
		got := fieldPath(tc.path)
		if got != tc.expected {
			t.Errorf("fieldPath(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}

func TestJoinPath(t *testing.T) {
	cases := []struct {
		path, key, expected string
	}{
		{"", "a", "a"},
		{"a", "b", "a.b"},
		{"a.b", "c", "a.b.c"},
		{"", "", ""},
	}

	for _, tc := range cases {
		got := joinPath(tc.path, tc.key)
		if got != tc.expected {
			t.Errorf("joinPath(%q, %q) = %q, want %q", tc.path, tc.key, got, tc.expected)
		}
	}
}