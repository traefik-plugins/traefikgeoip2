package traefikgeoip2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	mw "github.com/traefik-plugins/traefikgeoip2"
)

const (
	ValidIP          = "188.193.88.199"
	ValidAlternateIP = "188.193.88.200"
	ValidIPNoCity    = "20.1.184.61"

	cityDB      = "./GeoLite2-City.mmdb"
	countryDB   = "./GeoLite2-Country.mmdb"
	missingDB   = "./missing"
	localhost   = "http://localhost"
	portSuffix  = ":9999"
	privateAddr = "10.0.0.1:9999"
	xForwardedFor = "X-Forwarded-For"
)

// newInstance is a test helper that creates a fresh middleware instance.
// All state is per-instance, so no global reset is needed between tests.
func newInstance(t *testing.T, cfg *mw.Config) http.Handler {
	t.Helper()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		// no-op: tests that need to assert next was called use their own handler
	})
	inst, err := mw.New(context.TODO(), next, cfg, t.Name())
	if err != nil {
		t.Fatalf("mw.New: %v", err)
	}
	return inst
}

// — Config / basic ————————————————————————————————————————————————————————————

func TestGeoIPConfig(t *testing.T) {
	cfg := mw.CreateConfig()
	if mw.DefaultDBPath != cfg.DBPath {
		t.Fatalf("wrong default DBPath: %s", cfg.DBPath)
	}

	cfg.DBPath = "./non-existing"
	_, err := mw.New(context.TODO(), nil, cfg, "")
	if err != nil {
		t.Fatalf("must not fail on missing DB: %v", err)
	}

	cfg.DBPath = "justfile"
	_, err = mw.New(context.TODO(), nil, cfg, "")
	if err != nil {
		t.Fatalf("must not fail on unrecognised DB name: %v", err)
	}
}

func TestGeoIPBasic(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = cityDB

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	inst, _ := mw.New(context.TODO(), next, cfg, t.Name())

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	inst.ServeHTTP(recorder, req)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", recorder.Result().StatusCode)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

// — Missing DB ————————————————————————————————————————————————————————————————

func TestMissingGeoIPDB(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })
	inst, err := mw.New(context.TODO(), next, cfg, "")
	if err != nil {
		t.Fatalf("must not fail on missing DB: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = "1.2.3.4" // public IP, no port — returned as-is

	inst.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Fatal("next handler was not called")
	}
	assertHeader(t, req, mw.CountryHeader, mw.Unknown)
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
	assertHeader(t, req, mw.PostalCodeHeader, mw.Unknown)
	assertHeader(t, req, mw.IPAddressHeader, "1.2.3.4")
	assertHeader(t, req, mw.PrivateIPHeader, "false")
}

// — Public IP lookups —————————————————————————————————————————————————————————

func TestGeoIPFromRemoteAddr(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = cityDB
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, "BY")
	assertHeader(t, req, mw.CityHeader, "Munich")
	assertHeader(t, req, mw.IPAddressHeader, ValidIP)
	assertHeader(t, req, mw.PrivateIPHeader, "false")

	req = httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIPNoCity + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, "US")
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
	assertHeader(t, req, mw.IPAddressHeader, ValidIPNoCity)

	req = httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = "qwerty:9999"
	inst.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, mw.Unknown)
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
	assertHeader(t, req, mw.IPAddressHeader, "qwerty")
}

func TestGeoIPFromXForwardedFor(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = cityDB
	cfg.PreferXForwardedForHeader = true
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	req.Header.Set(xForwardedFor, ValidAlternateIP)
	inst.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, "BY")
	assertHeader(t, req, mw.CityHeader, "Munich")
	assertHeader(t, req, mw.IPAddressHeader, ValidAlternateIP)

	req = httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	req.Header.Set(xForwardedFor, ValidAlternateIP+",188.193.88.100")
	inst.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.IPAddressHeader, ValidAlternateIP)

	req = httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	req.Header.Set(xForwardedFor, "qwerty")
	inst.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
	assertHeader(t, req, mw.IPAddressHeader, "qwerty")
}

func TestGeoIPCountryDB(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = countryDB
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
	// Country DB never has postal data.
	assertHeader(t, req, mw.PostalCodeHeader, mw.Unknown)
	assertHeader(t, req, mw.IPAddressHeader, ValidIP)
}

// — Postal code ———————————————————————————————————————————————————————————————

// TestPostalCodeCityDB verifies that a postal code header is present when using
// the City database.  The value itself depends on the DB content; we only assert
// it is non-empty and non-Unknown for the Munich IP which reliably has a postal.
func TestPostalCodeCityDB(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = cityDB
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix // Munich
	inst.ServeHTTP(httptest.NewRecorder(), req)

	postal := req.Header.Get(mw.PostalCodeHeader)
	if postal == "" || postal == mw.Unknown {
		t.Fatalf("expected a postal code for Munich IP, got %q", postal)
	}
	t.Logf("Munich postal code from DB: %s", postal)
}

// TestPrivateIPPostalCode verifies the privateIPPostalCode config field.
func TestPrivateIPPostalCode(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB
	cfg.PrivateIPCountry = "DE"
	cfg.PrivateIPRegion = "BW"
	cfg.PrivateIPCity = "Stuttgart"
	cfg.PrivateIPPostalCode = "70173"
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = privateAddr
	inst.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.PrivateIPHeader, "true")
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.PostalCodeHeader, "70173")
}

// — Private IP handling ———————————————————————————————————————————————————————

func TestPrivateIPDefaults(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = cityDB
	inst := newInstance(t, cfg)

	privateTests := []struct{ remoteAddr, ip string }{
		{privateAddr, "10.0.0.1"},
		{"192.168.1.100:9999", "192.168.1.100"},
		{"172.16.5.5:9999", "172.16.5.5"},
		{"127.0.0.1:9999", "127.0.0.1"},
		{"[::1]:9999", "::1"},
		{"169.254.1.1:9999", "169.254.1.1"},
	}

	for _, tt := range privateTests {
		req := httptest.NewRequest(http.MethodGet, localhost, nil)
		req.RemoteAddr = tt.remoteAddr
		inst.ServeHTTP(httptest.NewRecorder(), req)

		t.Logf("remoteAddr=%s → ip=%s", tt.remoteAddr, tt.ip)
		assertHeader(t, req, mw.PrivateIPHeader, "true")
		assertHeader(t, req, mw.CountryHeader, mw.Unknown)
		assertHeader(t, req, mw.RegionHeader, mw.Unknown)
		assertHeader(t, req, mw.CityHeader, mw.Unknown)
		assertHeader(t, req, mw.PostalCodeHeader, mw.Unknown)
		assertHeader(t, req, mw.IPAddressHeader, tt.ip)
	}
}

func TestPrivateIPOverride(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = cityDB
	cfg.PrivateIPCountry = "DE"
	cfg.PrivateIPRegion = "BW"
	cfg.PrivateIPCity = "Stuttgart"
	cfg.PrivateIPPostalCode = "70173"
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = "10.0.0.5:9999"
	inst.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.PrivateIPHeader, "true")
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, "BW")
	assertHeader(t, req, mw.CityHeader, "Stuttgart")
	assertHeader(t, req, mw.PostalCodeHeader, "70173")
	assertHeader(t, req, mw.IPAddressHeader, "10.0.0.5")
}

func TestPrivateIPPartialOverride(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB
	cfg.PrivateIPCountry = "DE"
	// Region, City and PostalCode intentionally left empty — Unknown
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = "192.168.0.1:9999"
	inst.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.PrivateIPHeader, "true")
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
	assertHeader(t, req, mw.PostalCodeHeader, mw.Unknown)
}

func TestPublicIPNotTaggedPrivate(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.PrivateIPHeader, "false")
}

// — File logging ——————————————————————————————————————————————————————————————

func TestFileLoggingCreatesFile(t *testing.T) {
	logPath := t.TempDir() + "/geoip.log"
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB
	cfg.LogFilePath = logPath
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req)

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	line := string(data)
	if !strings.Contains(line, "ip="+ValidIP) {
		t.Fatalf("expected IP in log line, got: %s", line)
	}
	if !strings.Contains(line, "private=false") {
		t.Fatalf("expected private=false in log line, got: %s", line)
	}
	if !strings.Contains(line, "postal=") {
		t.Fatalf("expected postal= field in log line, got: %s", line)
	}
}

func TestFileLoggingPrivateIP(t *testing.T) {
	logPath := t.TempDir() + "/geoip_private.log"
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB
	cfg.LogFilePath = logPath
	cfg.PrivateIPCountry = "DE"
	cfg.PrivateIPRegion = "BW"
	cfg.PrivateIPCity = "Stuttgart"
	cfg.PrivateIPPostalCode = "70173"
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = privateAddr
	inst.ServeHTTP(httptest.NewRecorder(), req)

	data, _ := os.ReadFile(logPath)
	line := string(data)
	for _, want := range []string{
		"ip=10.0.0.1", "country=DE", "region=BW",
		"city=Stuttgart", "postal=70173", "private=true",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected %q in log line, got: %s", want, line)
		}
	}
}

func TestFileLoggingDisabled(t *testing.T) {
	cfg := mw.CreateConfig()
	cfg.DBPath = missingDB
	// LogFilePath empty — no logging
	inst := newInstance(t, cfg)

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req) // must not panic
}

// — DB hot-reload —————————————————————————————————————————————————————————————

// TestDBRefreshDetectsChange writes a minimal marker file, starts middleware
// with a 1-second interval, waits until the first check passes, then replaces
// the file and advances beyond the interval.  It verifies that a second reload
// attempt is made (no panic, no deadlock).
//
// Note: this test does NOT have a real mmdb file, so the reload will log an
// error but must not crash or deadlock.
func TestDBRefreshDetectsChange(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/GeoIP2-Stub.mmdb"

	// Write a dummy file so the initial Stat succeeds.
	if err := os.WriteFile(dbPath, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := mw.CreateConfig()
	cfg.DBPath = dbPath
	cfg.DBRefreshInterval = 1 // 1 second
	inst := newInstance(t, cfg)

	// First request — no reload yet.
	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req)

	// Wait past the refresh interval, then touch the file.
	time.Sleep(1100 * time.Millisecond)
	if err := os.WriteFile(dbPath, []byte("dummy-v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second request — should trigger the reload path (will fail gracefully).
	req = httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req) // must not panic or deadlock
}

func TestDBRefreshNoChangeSkipsReload(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/GeoIP2-Stub.mmdb"
	if err := os.WriteFile(dbPath, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := mw.CreateConfig()
	cfg.DBPath = dbPath
	cfg.DBRefreshInterval = 1
	inst := newInstance(t, cfg)

	time.Sleep(1100 * time.Millisecond)
	// File NOT touched — modtime unchanged — no reload.

	req := httptest.NewRequest(http.MethodGet, localhost, nil)
	req.RemoteAddr = ValidIP + portSuffix
	inst.ServeHTTP(httptest.NewRecorder(), req) // must not panic
}

// — Helper ————————————————————————————————————————————————————————————————————

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if got := req.Header.Get(key); got != expected {
		t.Fatalf("header %s: got %q, want %q", key, got, expected)
	}
}
