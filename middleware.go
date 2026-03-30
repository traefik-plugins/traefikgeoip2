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
	"sync"
	"time"

	"github.com/IncSW/geoip2"
)

// ResetLookup is kept for backwards compatibility with existing tests.
// The lookup function is now per-instance; each New() call creates a fresh state.
func ResetLookup() {
	// no-op: lookup state is no longer global
}

// Config the plugin configuration.
type Config struct {
	// DBPath is the path to the MaxMind GeoIP2/GeoLite2 .mmdb database file.
	DBPath string `json:"dbPath,omitempty"`

	// PreferXForwardedForHeader uses the first IP in X-Forwarded-For instead of RemoteAddr.
	// Enable this when Traefik sits behind a proxy (e.g. Cloudflare).
	PreferXForwardedForHeader bool `json:"preferXForwardedForHeader,omitempty"`

	// PrivateIPCountry is the ISO 3166-1 alpha-2 country code returned for private/LAN IPs.
	// If empty, Unknown ("XX") is used.  Example: "DE"
	PrivateIPCountry string `json:"privateIPCountry,omitempty"`

	// PrivateIPRegion is the ISO 3166-2 region/subdivision code returned for private/LAN IPs.
	// If empty, Unknown ("XX") is used.  Example: "BW"
	PrivateIPRegion string `json:"privateIPRegion,omitempty"`

	// PrivateIPCity is the city name returned for private/LAN IPs.
	// If empty, Unknown ("XX") is used.  Example: "Stuttgart"
	PrivateIPCity string `json:"privateIPCity,omitempty"`

	// PrivateIPPostalCode is the postal/ZIP code returned for private/LAN IPs.
	// If empty, Unknown ("XX") is used.  Example: "70173"
	PrivateIPPostalCode string `json:"privateIPPostalCode,omitempty"`

	// DBRefreshInterval is how often (in seconds) Traefik checks whether the mmdb file
	// has been modified on disk and reloads it if so.
	// 0 disables automatic refresh (default).  Example: 3600 (= every hour)
	DBRefreshInterval int `json:"dbRefreshInterval,omitempty"`

	// LogFilePath enables per-request GeoIP logging to a file.
	// Each line: <RFC3339> ip=<ip> country=<cc> region=<rc> city=<city> postal=<zip> private=<bool>
	// If empty, file logging is disabled; Traefik's own log still receives error messages.
	LogFilePath string `json:"logFilePath,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DBPath: DefaultDBPath,
	}
}

// TraefikGeoIP2 a traefik geoip2 plugin.
type TraefikGeoIP2 struct {
	next                      http.Handler
	name                      string
	preferXForwardedForHeader bool
	privateIPCountry          string
	privateIPRegion           string
	privateIPCity             string
	privateIPPostalCode       string
	logger                    *log.Logger

	// DB state — all fields below are protected by mu.
	mu                sync.RWMutex
	lookup            LookupGeoIP2
	dbPath            string
	dbModTime         time.Time
	lastChecked       time.Time
	dbRefreshInterval time.Duration
}

// New created a new TraefikGeoIP2 plugin.
func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	// --- File logger setup -----------------------------------------------
	var fileLogger *log.Logger
	if cfg.LogFilePath != "" {
		f, err := os.OpenFile(cfg.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("[geoip2] Cannot open log file: path=%s, name=%s, err=%v",
				cfg.LogFilePath, name, err)
		} else {
			fileLogger = log.New(f, "", 0)
			log.Printf("[geoip2] File logging enabled: path=%s, name=%s", cfg.LogFilePath, name)
		}
	}

	mwInstance := &TraefikGeoIP2{
		next:                      next,
		name:                      name,
		preferXForwardedForHeader: cfg.PreferXForwardedForHeader,
		privateIPCountry:          cfg.PrivateIPCountry,
		privateIPRegion:           cfg.PrivateIPRegion,
		privateIPCity:             cfg.PrivateIPCity,
		privateIPPostalCode:       cfg.PrivateIPPostalCode,
		logger:                    fileLogger,
		dbPath:                    cfg.DBPath,
		dbRefreshInterval:         time.Duration(cfg.DBRefreshInterval) * time.Second,
	}

	// --- Initial DB load -------------------------------------------------
	lkup, modTime, err := loadDB(cfg.DBPath)
	if err != nil {
		log.Printf("[geoip2] DB init failed (requests will use Unknown): db=%s, name=%s, err=%v",
			cfg.DBPath, name, err)
		return mwInstance, nil
	}
	mwInstance.lookup = lkup
	mwInstance.dbModTime = modTime
	mwInstance.lastChecked = time.Now()
	log.Printf("[geoip2] DB loaded: db=%s, name=%s, modtime=%s",
		cfg.DBPath, name, modTime.Format(time.RFC3339))

	return mwInstance, nil
}

// ServeHTTP implements http.Handler.
func (mw *TraefikGeoIP2) ServeHTTP(reqWr http.ResponseWriter, req *http.Request) {
	// Hot-reload check (no-op when dbRefreshInterval == 0).
	if mw.dbRefreshInterval > 0 {
		mw.maybeReloadDB()
	}

	ipStr := getClientIP(req, mw.preferXForwardedForHeader)
	private := isPrivateIP(ipStr)

	var res *GeoIPResult

	switch {
	case private:
		// Private / LAN / loopback: skip DB, use operator-configured values.
		res = &GeoIPResult{
			country:    coalesce(mw.privateIPCountry, Unknown),
			region:     coalesce(mw.privateIPRegion, Unknown),
			city:       coalesce(mw.privateIPCity, Unknown),
			postalCode: coalesce(mw.privateIPPostalCode, Unknown),
		}

	default:
		mw.mu.RLock()
		currentLookup := mw.lookup
		mw.mu.RUnlock()

		if currentLookup == nil {
			res = &GeoIPResult{country: Unknown, region: Unknown, city: Unknown, postalCode: Unknown}
		} else {
			var err error
			res, err = currentLookup(net.ParseIP(ipStr))
			if err != nil {
				log.Printf("[geoip2] lookup failed: ip=%s, err=%v", ipStr, err)
				res = &GeoIPResult{country: Unknown, region: Unknown, city: Unknown, postalCode: Unknown}
			}
		}
	}

	req.Header.Set(CountryHeader, res.country)
	req.Header.Set(RegionHeader, res.region)
	req.Header.Set(CityHeader, res.city)
	req.Header.Set(PostalCodeHeader, res.postalCode)
	req.Header.Set(IPAddressHeader, ipStr)
	req.Header.Set(PrivateIPHeader, boolToStr(private))

	mw.logEntry(ipStr, res, private)
	mw.next.ServeHTTP(reqWr, req)
}

// maybeReloadDB checks whether the mmdb file has been modified since the last
// load and, if so, transparently swaps in a new reader.
// Uses double-checked locking to avoid redundant reloads under concurrent traffic.
func (mw *TraefikGeoIP2) maybeReloadDB() {
	// Fast path: read-lock to check the interval without blocking normal lookups.
	mw.mu.RLock()
	due := time.Since(mw.lastChecked) >= mw.dbRefreshInterval
	mw.mu.RUnlock()
	if !due {
		return
	}

	// Slow path: exclusive lock for the actual reload.
	mw.mu.Lock()
	defer mw.mu.Unlock()

	// Double-check: another goroutine may have already reloaded while we waited.
	if time.Since(mw.lastChecked) < mw.dbRefreshInterval {
		return
	}
	// Always update lastChecked so we don't busy-loop on a broken file.
	mw.lastChecked = time.Now()

	info, err := os.Stat(mw.dbPath)
	if err != nil {
		log.Printf("[geoip2] DB stat failed during refresh: path=%s, err=%v", mw.dbPath, err)
		return
	}
	if !info.ModTime().After(mw.dbModTime) {
		return // File unchanged.
	}

	newLookup, newModTime, err := loadDB(mw.dbPath)
	if err != nil {
		log.Printf("[geoip2] DB reload failed: path=%s, err=%v", mw.dbPath, err)
		return
	}
	mw.lookup = newLookup
	mw.dbModTime = newModTime
	log.Printf("[geoip2] DB reloaded: path=%s, modtime=%s", mw.dbPath, newModTime.Format(time.RFC3339))
}

// loadDB reads the mmdb at path, detects its type (City vs Country) from the
// filename, and returns an initialised LookupGeoIP2 together with the file's
// modification time.
//
// The underlying IncSW library can panic on malformed or truncated files.
// loadDB recovers from any such panic and returns it as a regular error so
// callers never have to worry about it.
func loadDB(dbPath string) (lkup LookupGeoIP2, modTime time.Time, retErr error) {
	// Catch panics from the mmdb parser (e.g. corrupt or truncated file).
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic while loading DB %s: %v", dbPath, r)
		}
	}()

	info, err := os.Stat(dbPath)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat %s: %w", dbPath, err)
	}
	modTime = info.ModTime()

	switch {
	case strings.Contains(dbPath, "City"):
		rdr, err := geoip2.NewCityReaderFromFile(dbPath)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("city reader %s: %w", dbPath, err)
		}
		return CreateCityDBLookup(rdr), modTime, nil

	case strings.Contains(dbPath, "Country"):
		rdr, err := geoip2.NewCountryReaderFromFile(dbPath)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("country reader %s: %w", dbPath, err)
		}
		return CreateCountryDBLookup(rdr), modTime, nil

	default:
		return nil, time.Time{}, fmt.Errorf("unknown DB type — filename must contain 'City' or 'Country': %s", dbPath)
	}
}

// logEntry appends one structured line to the file logger (no-op when disabled).
// Format: <RFC3339> ip=<ip> country=<cc> region=<rc> city=<city> postal=<zip> private=<bool>
func (mw *TraefikGeoIP2) logEntry(ip string, res *GeoIPResult, private bool) {
	if mw.logger == nil {
		return
	}
	mw.logger.Printf("%s ip=%s country=%s region=%s city=%s postal=%s private=%s",
		time.Now().UTC().Format(time.RFC3339),
		ip,
		res.country,
		res.region,
		res.city,
		res.postalCode,
		boolToStr(private),
	)
}

// isPrivateIP returns true for private, loopback, and link-local addresses.
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

// coalesce returns val if non-empty, otherwise fallback.
func coalesce(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func getClientIP(req *http.Request, preferXForwardedForHeader bool) string {
	if preferXForwardedForHeader {
		if fwd := req.Header.Get("X-Forwarded-For"); fwd != "" {
			return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
		}
	}
	remoteAddr := req.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteAddr = host
	}
	return remoteAddr
}
