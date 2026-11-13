package main

import "testing"

func TestSQLiteBool(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  any
	}{
		{"zero", int64(0), false},
		{"one", int64(1), true},
		{"bytes zero", []byte("0"), false},
		{"bytes one", []byte("1"), true},
		{"bool", true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := sqliteBool(test.input)
			if err != nil || got != test.want {
				t.Fatalf("sqliteBool(%#v) = %#v, %v; want %#v", test.input, got, err, test.want)
			}
		})
	}
}
