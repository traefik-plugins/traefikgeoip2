// Package traefikgeoip2 is a Traefik plugin for Maxmind GeoIP2.
package traefikgeoip2

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/IncSW/geoip2"
)

var lookup LookupGeoIP2

// ResetLookup reset lookup function.
func ResetLookup() {
	lookup = nil
}

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
	next http.Handler
	name string
}

// New created a new TraefikGeoIP2 plugin.
func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(cfg.DBPath); err != nil {
		log.Printf("[geoip2] DB not found: db=%s, name=%s, err=%v", cfg.DBPath, name, err)
		return &TraefikGeoIP2{
			next: next,
			name: name,
		}, nil
	}

	if lookup == nil && strings.Contains(cfg.DBPath, "City") {
		rdr, err := geoip2.NewCityReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("[geoip2] lookup DB is not initialized: db=%s, name=%s, err=%v", cfg.DBPath, name, err)
		} else {
			lookup = CreateCityDBLookup(rdr)
			log.Printf("[geoip2] lookup DB initialized: db=%s, name=%s, lookup=%v", cfg.DBPath, name, lookup)
		}
	}

	if lookup == nil && strings.Contains(cfg.DBPath, "Country") {
		rdr, err := geoip2.NewCountryReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("[geoip2] lookup DB is not initialized: db=%s, name=%s, err=%v", cfg.DBPath, name, err)
		} else {
			lookup = CreateCountryDBLookup(rdr)
			log.Printf("[geoip2] lookup DB initialized: db=%s, name=%s, lookup=%v", cfg.DBPath, name, lookup)
		}
	}

	return &TraefikGeoIP2{
		next: next,
		name: name,
	}, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(reqWr http.ResponseWriter, req *http.Request) {
	if lookup == nil {
		req.Header.Set(CountryHeader, Unknown)
		req.Header.Set(RegionHeader, Unknown)
		req.Header.Set(CityHeader, Unknown)
		req.Header.Set(IPAddressHeader, Unknown)
		mw.next.ServeHTTP(reqWr, req)
		return
	}

	ipStr := req.RemoteAddr
	tmp, _, err := net.SplitHostPort(ipStr)
	if err == nil {
		ipStr = tmp
	}

	res, err := lookup(net.ParseIP(ipStr))
	if err != nil {
		log.Printf("[geoip2] Unable to find: ip=%s, err=%v", ipStr, err)
		res = &GeoIPResult{
			country: Unknown,
			region:  Unknown,
			city:    Unknown,
		}
	}

	req.Header.Set(CountryHeader, res.country)
	req.Header.Set(RegionHeader, res.region)
	req.Header.Set(CityHeader, res.city)
	req.Header.Set(IPAddressHeader, ipStr)

	mw.next.ServeHTTP(reqWr, req)
}
