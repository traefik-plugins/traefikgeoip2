// Package traefikgeoip2 is a GeoIP2 plugin.
package traefikgeoip2

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/oschwald/geoip2-golang"
)

// DefaultDBPath default GeoIP2 database path.
const DefaultDBPath = "/var/lib/traefik-geoip2/GeoLite2-Country.mmdb"

const (
	// CountryHeader country header name.
	CountryHeader = "X-GeoIP2-Country"
	// RegionHeader region header name.
	RegionHeader = "X-GeoIP2-Region"
	// CityHeader city header name.
	CityHeader = "X-GeoIP2-City"
)

// Unknown constant for undefined data.
const Unknown = "XX"

// Config the plugin configuration.
type Config struct {
	DBPath string `json:"dbPath,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DBPath: DefaultDBPath,
	}
}

// TraefikGeoIP2 a traefik geoip2 plugin.
type TraefikGeoIP2 struct {
	next   http.Handler
	reader *geoip2.Reader
	name   string
}

// New created a new TraefikGeoIP2 plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(cfg.DBPath); err != nil {
		log.Printf("GeoIP DB not found: %s\n %v", cfg.DBPath, err)
		return nil, fmt.Errorf("geoip db not found: %s %w", cfg.DBPath, err)
	}

	rdr, err := geoip2.Open(cfg.DBPath)
	if err != nil {
		log.Printf("GeoIP DB %s not initialized: %v", cfg.DBPath, err)
		return nil, fmt.Errorf("geoip db %s not initialized: %w", cfg.DBPath, err)
	}

	return &TraefikGeoIP2{
		reader: rdr,
		next:   next,
		name:   name,
	}, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Printf("@@@@ addr: %v, realIp: %v, xForwFor: %v",
		req.RemoteAddr, req.Header.Get("X-Real-Ip"), req.Header.Get("X-Forwarded-For"))

	country := Unknown
	region := Unknown
	city := Unknown

	ip := net.ParseIP(req.RemoteAddr)
	if ip != nil {
		rec, err := mw.reader.City(ip)
		if err != nil {
			log.Printf("Error retrieving GeoIP for %v, %v", ip, err)
		} else {
			country = rec.Country.IsoCode
			if mw.reader.Metadata().DatabaseType == "GeoLite2-Country" {
				region = Unknown
				city = Unknown
			} else {
				city = rec.City.Names["en"]
				if rec.Subdivisions != nil {
					region = rec.Subdivisions[0].Names["en"]
				}
			}
		}
	}

	req.Header.Set(CountryHeader, country)
	req.Header.Set(RegionHeader, region)
	req.Header.Set(CityHeader, city)

	mw.next.ServeHTTP(rw, req)
}
