package service

import (
	"testing"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

func TestGeoLocationDisplayName(t *testing.T) {
	for _, tc := range []struct {
		name string
		loc  model.Session
		want string
	}{
		{"city region country", model.Session{City: "Berlin", Region: "BE", Country: "DE"}, "Berlin, BE, DE"},
		{"country only", model.Session{Country: "IR"}, "IR"},
		{"unknown", model.Session{}, "Unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := GeoLocationDisplayName(tc.loc); got != tc.want {
				t.Fatalf("display name = %q, want %q", got, tc.want)
			}
		})
	}
}
