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
	next          http.Handler
	cityReader    *geoip2.CityReader
	countryReader *geoip2.CountryReader
	name          string
}

// New created a new TraefikGeoIP2 plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if _, err := os.Stat(cfg.DBPath); err != nil {
		log.Printf("GeoIP DB not found: %s\n %v", cfg.DBPath, err)
		return nil, fmt.Errorf("geoip db not found: %s %w", cfg.DBPath, err)
	}

	if strings.Contains(cfg.DBPath, "City") {
		cityReader, err := geoip2.NewCityReaderFromFile(cfg.DBPath)
		if err != nil {
			log.Printf("GeoIP DB %s not initialized: %v", cfg.DBPath, err)
			return nil, fmt.Errorf("geoip db %s not initialized: %w", cfg.DBPath, err)
		}
		return &TraefikGeoIP2{
			countryReader: nil,
			cityReader:    cityReader,
			next:          next,
			name:          name,
		}, nil
	}

	countryReader, err := geoip2.NewCountryReaderFromFile(cfg.DBPath)
	if err != nil {
		log.Printf("GeoIP DB %s not initialized: %v", cfg.DBPath, err)
		return nil, fmt.Errorf("geoip db %s not initialized: %w", cfg.DBPath, err)
	}
	return &TraefikGeoIP2{
		countryReader: countryReader,
		cityReader:    nil,
		next:          next,
		name:          name,
	}, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Printf("@@@@ remoteAddr: %v, xRealIp: %v", req.RemoteAddr, req.Header.Get(RealIPHeader))

	retval := GeoIPResult{
		country: Unknown,
		region:  Unknown,
		city:    Unknown,
	}

	ipStr := req.Header.Get(RealIPHeader)
	if ipStr == "" {
		ipStr = req.RemoteAddr
	}

	ip := net.ParseIP(ipStr)
	if ip != nil && mw.cityReader != nil {
		rec, err := mw.cityReader.Lookup(ip)
		if err != nil {
			log.Printf("Error retrieving GeoIP for %v, %v", ip, err)
		} else {
			retval.country = rec.Country.ISOCode
			retval.city = rec.City.Names["en"]
			if rec.Subdivisions != nil {
				retval.region = rec.Subdivisions[0].Names["en"]
			}
		}
	} else if ip != nil && mw.countryReader != nil {
		rec, err := mw.countryReader.Lookup(ip)
		if err != nil {
			log.Printf("Error retrieving GeoIP for %v, %v", ip, err)
		} else {
			retval.country = rec.Country.ISOCode
		}
	}

	ApplyGeoIPResult(req, &retval)

	mw.next.ServeHTTP(rw, req)
}
