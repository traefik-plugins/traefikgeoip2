package traefikgeoip2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mw "github.com/GiGInnovationLabs/traefikgeoip2"
)

const (
	ValidIP        = "188.193.88.199"
	ValidIPAndPort = "188.193.88.199:9999"
)

func TestGeoIPConfig(t *testing.T) {
	mwCfg := mw.CreateConfig()
	if mw.DefaultDBPath != mwCfg.DBPath {
		t.Fatalf("Incorrect path")
	}

	mwCfg.DBPath = "./non-existing"
	_, err := mw.New(context.TODO(), nil, mwCfg, "")
	if err == nil || !strings.Contains(err.Error(), "./non-existing") {
		t.Fatalf("Error is empty or incorrect %v", err)
	}

	mwCfg.DBPath = "Makefile"
	_, err = mw.New(context.TODO(), nil, mwCfg, "")
	if err.Error() != "db `Makefile' not initialized: invalid metadata type: 3" {
		t.Fatalf("Incorrect error: %v", err)
	}
}

func TestGeoIPBasic(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-City.mmdb"

	called := false
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { called = true })

	instance, err := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")
	if err != nil {
		t.Fatalf("Error creating %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

	instance.ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("Invalid return code")
	}
	if called != true {
		t.Fatalf("next handler was not called")
	}
}

func TestGeoIPFromRemoteAddr(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-City.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = ValidIPAndPort
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, "Bavaria")
	assertHeader(t, req, mw.CityHeader, "Munich")

	req = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "qwerty:9999"
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, mw.Unknown)
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
}

func TestGeoIPCountryDBFromRemoteAddr(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-Country.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.TODO(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = ValidIPAndPort
	instance.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
}

func TestGeoIPFromXRealIP(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-City.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.Background(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "1.1.1.1:9999"
	req.Header.Set("X-Real-Ip", ValidIP)

	instance.ServeHTTP(httptest.NewRecorder(), req)
	assertHeader(t, req, mw.CountryHeader, "DE")
	assertHeader(t, req, mw.RegionHeader, "Bavaria")
	assertHeader(t, req, mw.CityHeader, "Munich")
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if req.Header.Get(key) != expected {
		t.Fatalf("invalid value of header [%s] != %s", key, req.Header.Get(key))
	}
}
