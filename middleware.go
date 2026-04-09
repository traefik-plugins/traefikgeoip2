// Package traefikgeoip2 is a Traefik plugin for Maxmind GeoIP2.
package traefikgeoip2

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"slices"
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
	DBPath                    string   `json:"dbPath,omitempty"`
	PreferXForwardedForHeader bool     `json:"preferXForwardedForHeader,omitempty"`
	DetectionMode             string   `json:"detectionMode,omitempty"`
	CountryCodes              []string `json:"countryCodes,omitempty"`
	SpecificIPList            []string `json:"specificIpList,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DBPath:                    DefaultDBPath,
		DetectionMode:             DetectionModeAllow, // Default to ALLOW to forbid all requests if no set countryCodes array.
		PreferXForwardedForHeader: false,
		CountryCodes:              make([]string, 0),
		SpecificIPList:            make([]string, 0),
	}
}

// TraefikGeoIP2 a traefik geoip2 plugin.
type TraefikGeoIP2 struct {
	next                      http.Handler
	name                      string
	preferXForwardedForHeader bool
	detectionMode             string
	countryCodes              []string
	specificIPList            []string
}

// New created a new TraefikGeoIP2 plugin.
func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	_, err := os.Stat(cfg.DBPath)
	if err != nil {
		log.Printf("[geoip2] DB not found: db=%s, name=%s, err=%v", cfg.DBPath, name, err)

		return &TraefikGeoIP2{
			next:                      next,
			name:                      name,
			preferXForwardedForHeader: cfg.PreferXForwardedForHeader,
			detectionMode:             cfg.DetectionMode,
			countryCodes:              cfg.CountryCodes,
			specificIPList:            cfg.SpecificIPList,
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

	if len(cfg.CountryCodes) > 0 {
		for _, countryCode := range cfg.CountryCodes {
			if reflect.TypeOf(countryCode).String() != "string" {
				log.Printf(
					"[geoip2] Bad type of countryCode: CountryCode=%v, type=%s", countryCode,
					reflect.TypeOf(countryCode).String(),
				)
			} else if countryCode == "" {
				log.Printf("[geoip2] No empty countryCode allowed")
			}
		}
	}

	if len(cfg.SpecificIPList) > 0 {
		for _, specificIP := range cfg.SpecificIPList {
			if reflect.TypeOf(specificIP).String() != "string" {
				log.Printf(
					"[geoip2] Bad type of specificIP: SpecificIP=%v, type=%s", specificIP,
					reflect.TypeOf(specificIP).String(),
				)
			} else if specificIP == "" {
				log.Println("[geoip2] No empty specificIP allowed")
			} else if net.ParseIP(specificIP) == nil {
				if strings.Contains(specificIP, "/") {
					_, _, err := net.ParseCIDR(specificIP)
					if err != nil {
						log.Printf("[geoip2] Bad range of specificIP: CIDR=%s", specificIP)
					}
				} else {
					log.Printf("[geoip2] Bad specificIP set %s", specificIP)
				}
			}
		}
	}

	traefikGeoIP2 := &TraefikGeoIP2{
		next:                      next,
		name:                      name,
		preferXForwardedForHeader: cfg.PreferXForwardedForHeader,
		detectionMode:             cfg.DetectionMode,
		countryCodes:              cfg.CountryCodes,
		specificIPList:            cfg.SpecificIPList,
	}
	return traefikGeoIP2, nil
}

func (mw *TraefikGeoIP2) ServeHTTP(reqWr http.ResponseWriter, req *http.Request) {
	if lookup == nil {
		log.Println("[geoip2] Unable to get lookup")
		switch mw.detectionMode {
		case DetectionModeAllow:
			reqWr.WriteHeader(http.StatusForbidden)
		case DetectionModeDeny:
			req.Header.Set(CountryHeader, Unknown)
			req.Header.Set(RegionHeader, Unknown)
			req.Header.Set(CityHeader, Unknown)
			req.Header.Set(IPAddressHeader, Unknown)

			mw.next.ServeHTTP(reqWr, req)
		default:
			reqWr.WriteHeader(http.StatusForbidden)
		}

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

	found := false
	if slices.Contains(mw.countryCodes, res.country) {
		found = true
	}

	if checkIPIntoList(mw.specificIPList, ipStr) {
		found = true
	}

	switch mw.detectionMode {
	case DetectionModeAllow:
		if found {
			log.Printf("[geoip2] Serve detectionMode=%s, ip=%s, country=%v", mw.detectionMode, ipStr, res.country)
			mw.next.ServeHTTP(reqWr, req)
		} else {
			log.Printf("[geoip2] Forbid detectionMode=%s, ip=%s, country=%v", mw.detectionMode, ipStr, res.country)
			reqWr.WriteHeader(http.StatusForbidden)
		}
	case DetectionModeDeny:
		if found {
			log.Printf("[geoip2] Forbid detectionMode=%s, ip=%s, country=%v", mw.detectionMode, ipStr, res.country)
			reqWr.WriteHeader(http.StatusForbidden)
		} else {
			log.Printf("[geoip2] Serve detectionMode=%s, ip=%s, country=%v", mw.detectionMode, ipStr, res.country)
			mw.next.ServeHTTP(reqWr, req)
		}
	default:
		reqWr.WriteHeader(http.StatusForbidden)
	}
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

func checkIPIntoList(ipList []string, clientIP string) bool {
	for _, ip := range ipList {
		if net.ParseIP(ip).Equal(net.ParseIP(clientIP)) {
			return true
		} else if checkIPIntoRange(ip, clientIP) {
			return true
		}
	}

	return false
}

func checkIPIntoRange(ipRange, clientIP string) bool {
	_, subnet, err := net.ParseCIDR(ipRange)
	if err != nil {
		return false
	}
	ip := net.ParseIP(clientIP)
	if len(ip) == 0 {
		return false
	}

	return subnet.Contains(ip)
}
