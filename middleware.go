// Package traefikgeoip2 is a Traefik plugin for Maxmind GeoIP2.
package traefikgeoip2

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IncSW/geoip2"
)

var lookup LookupGeoIP2

var ticker *time.Ticker

// ResetLookup reset lookup function.
func ResetLookup() {
	lookup = nil
}

// Config the plugin configuration.
type Config struct {
	DBPath                    string `json:"dbPath,omitempty"`
	PreferXForwardedForHeader bool
	RefreshInterval           string `json:"refreshInterval"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DBPath:          DefaultDBPath,
		RefreshInterval: DefaultRefreshInterval,
	}
}

// TraefikGeoIP2 a traefik geoip2 plugin.
type TraefikGeoIP2 struct {
	next                      http.Handler
	name                      string
	preferXForwardedForHeader bool
	refreshInterval           string
	dbPath                    string
}

// New created a new TraefikGeoIP2 plugin.
func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {

	if cfg.RefreshInterval != "0" {
		// when change the dynamic config need reload geoDB
		ResetLookup()
		log.Printf("[geoip2] DB refreshInterval: refreshInterval=%s", cfg.RefreshInterval)
	}
	if _, err := os.Stat(cfg.DBPath); err != nil {
		log.Printf("[geoip2] DB not found: db=%s, name=%s, err=%v", cfg.DBPath, name, err)
		return &TraefikGeoIP2{
			next:                      next,
			name:                      name,
			preferXForwardedForHeader: cfg.PreferXForwardedForHeader,
			refreshInterval:           cfg.RefreshInterval,
			dbPath:                    cfg.DBPath,
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
	geo := &TraefikGeoIP2{
		next:                      next,
		name:                      name,
		preferXForwardedForHeader: cfg.PreferXForwardedForHeader,
		refreshInterval:           cfg.RefreshInterval,
		dbPath:                    cfg.DBPath,
	}
	//disable refresh
	if cfg.RefreshInterval != "0" {
		go geo.refreshDB()
	} else {
		if ticker != nil {
			//stop old ticker when change config to "0" from other value
			ticker.Stop()
		}
	}
	return geo, nil
	//return &TraefikGeoIP2{
	//	next:                      next,
	//	name:                      name,
	//	preferXForwardedForHeader: cfg.PreferXForwardedForHeader,
	//}, nil
}

func (g *TraefikGeoIP2) refreshDB() {
	if ticker != nil {
		//stop old ticker
		ticker.Stop()
	}
	interval, err := time.ParseDuration(g.refreshInterval)
	if err != nil {
		log.Printf("[geoip2] Config refreshInterval error , has been replaced with '1h': refreshInterval=%s, err=%v", g.refreshInterval, err)
		interval, err = time.ParseDuration("1h")
	}
	ticker = time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if strings.Contains(g.dbPath, "City") {
				rdr, err := geoip2.NewCityReaderFromFile(g.dbPath)
				if err != nil {
					log.Printf("[geoip2] lookup DB is not initialized: db=%s, name=%s, err=%v", g.dbPath, err)
				} else {
					lookup = CreateCityDBLookup(rdr)
					log.Printf("[geoip2] lookup DB initialized: db=%s, name=%s, lookup=%v", g.dbPath, lookup)
				}
			}
			if strings.Contains(g.dbPath, "Country") {
				rdr, err := geoip2.NewCountryReaderFromFile(g.dbPath)
				if err != nil {
					log.Printf("Error refreshing GeoIP database: %s", err)
					continue
				} else {
					lookup = CreateCountryDBLookup(rdr)
					log.Printf("[geoip2] lookup DB initialized: db=%s, lookup=%v", g.dbPath, lookup)
				}
			}
			log.Println("GeoIP database refreshed successfully")
		}
	}
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

	ipStr := getClientIP(req, mw.preferXForwardedForHeader)
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

func getClientIP(req *http.Request, preferXForwardedForHeader bool) string {
	if preferXForwardedForHeader {
		// Check X-Forwarded-For header first
		forwardedFor := req.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			ips := strings.Split(forwardedFor, ",")
			return strings.TrimSpace(ips[0])
		}
	}

	// If X-Forwarded-For is not present or retrieval is not enabled, fallback to RemoteAddr
	remoteAddr := req.RemoteAddr
	tmp, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = tmp
	}
	return remoteAddr
}
