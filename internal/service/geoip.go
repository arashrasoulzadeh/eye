package service

import (
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/arashrasoulzadeh/eye/internal/model"
	"github.com/oschwald/maxminddb-golang"
	"gorm.io/gorm"
)

// GeoIPRecord is the subset of GeoLite2 City data persisted by Eye.
type GeoIPRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		Names   map[string]string `maxminddb:"names"`
		ISOCode string            `maxminddb:"iso_code"`
	} `maxminddb:"subdivisions"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
}

// GeoIPService performs local-only lookups. A missing database is a supported
// configuration: sessions remain usable and simply have unknown location.
type GeoIPService struct {
	DB     *gorm.DB
	Reader *maxminddb.Reader
	mu     sync.RWMutex
}

func OpenGeoIP(path string) (*GeoIPService, error) {
	service := &GeoIPService{}
	if strings.TrimSpace(path) == "" {
		return service, nil
	}
	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	service.Reader = reader
	return service, nil
}

func (s *GeoIPService) Close() error {
	if s == nil || s.Reader == nil {
		return nil
	}
	return s.Reader.Close()
}

func (s *GeoIPService) Lookup(ip string) (model.Session, bool) {
	if s == nil || s.Reader == nil {
		return model.Session{}, false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return model.Session{}, false
	}
	var record GeoIPRecord
	if err := s.Reader.Lookup(parsed, &record); err != nil {
		return model.Session{}, false
	}
	location := model.Session{Country: record.Country.ISOCode, Latitude: record.Location.Latitude, Longitude: record.Location.Longitude, Timezone: record.Location.TimeZone}
	if len(record.Subdivisions) > 0 {
		location.Region = record.Subdivisions[0].ISOCode
		if location.Region == "" {
			location.Region = record.Subdivisions[0].Names["en"]
		}
	}
	location.City = record.City.Names["en"]
	return location, true
}

func (s *GeoIPService) Enrich(session *model.Session) bool {
	if session == nil {
		return false
	}
	location, ok := s.Lookup(session.IP)
	if !ok {
		return false
	}
	session.Country, session.Region, session.City = location.Country, location.Region, location.City
	session.Latitude, session.Longitude, session.Timezone = location.Latitude, location.Longitude, location.Timezone
	return true
}

// Backfill enriches only sessions that do not already have a country, making
// it safe to rerun after downloading a newer database.
func (s *GeoIPService) Backfill(db *gorm.DB) (int64, error) {
	if s == nil || s.Reader == nil {
		return 0, nil
	}
	var sessions []model.Session
	if err := db.Where("country = '' AND ip <> ''").Find(&sessions).Error; err != nil {
		return 0, err
	}
	var updated int64
	for i := range sessions {
		if !s.Enrich(&sessions[i]) {
			continue
		}
		if err := db.Model(&sessions[i]).Select("country", "region", "city", "latitude", "longitude", "timezone").Updates(&sessions[i]).Error; err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

var ErrGeoIPUnavailable = errors.New("offline GeoIP database is not configured")

func GeoLocationDisplayName(session model.Session) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{session.City, session.Region, session.Country} {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, strings.TrimSpace(part))
		}
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, ", ")
}
