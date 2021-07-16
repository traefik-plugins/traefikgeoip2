// Package traefikgeoip2 is a Traefik plugin for Maxmind GeoIP2.
package traefikgeoip2

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/IncSW/geoip2"
)

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
	lookup LookupGeoIP2
	name   string
}

// New created a new TraefikGeoIP2 plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(cfg.DBPath); err != nil {
		log.Printf("GeoIP DB not found: %s\n %v", cfg.DBPath, err)
		return nil, fmt.Errorf("db `%s' not found: %w", cfg.DBPath, err)
	}

	retval := TraefikGeoIP2{
		lookup: nil,
		next:   next,
		name:   name,
	}

	if strings.Contains(cfg.DBPath, "City") {
		rdr, err := geoip2.NewCityReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("GeoIP DB %s not initialized: %v", cfg.DBPath, err)
			return nil, fmt.Errorf("db `%s' not initialized: %w", cfg.DBPath, err)
		}
		retval.lookup = CreateCityDBLookup(rdr)
	} else {
		rdr, err := geoip2.NewCountryReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("GeoIP DB %s not initialized: %v", cfg.DBPath, err)
			return nil, fmt.Errorf("db `%s' not initialized: %w", cfg.DBPath, err)
		}
		retval.lookup = CreateCountryDBLookup(rdr)
	}

	return &retval, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Printf("@@@@ remoteAddr: %v, xRealIp: %v", req.RemoteAddr, req.Header.Get(RealIPHeader))

	ipStr := req.Header.Get(RealIPHeader)
	if ipStr == "" {
		ipStr = req.RemoteAddr
		tmp, _, err := net.SplitHostPort(ipStr)
		if err == nil {
			ipStr = tmp
		}
	}

	res, err := mw.lookup(net.ParseIP(ipStr))
	if err != nil {
		log.Printf("Unable to find GeoIP data for `%s', %v", ipStr, err)
		res = &GeoIPResult{
			country: Unknown,
			region:  Unknown,
			city:    Unknown,
		}
	}

	ApplyGeoIPResult(req, res)

	mw.next.ServeHTTP(rw, req)
}
