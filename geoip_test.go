package traefikgeoip2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mw "github.com/GiGInnovationLabs/traefik-geoip2"
)

func TestGeoIPConfig(t *testing.T) {
	mwCfg := mw.CreateConfig()
	if mw.DefaultDBPath != mwCfg.DBPath {
		t.Fatalf("Incorrect path")
	}

	_, err := mw.New(context.TODO(), nil, mwCfg, "")
	if err == nil || !strings.Contains(err.Error(), mw.DefaultDBPath) {
		t.Fatalf("Error is empty or incorrect %v", err)
	}

	mwCfg.DBPath = "Makefile"
	_, err = mw.New(context.TODO(), nil, mwCfg, "")
	if err.Error() != "geoip db Makefile not initialized: invalid metadata type: 3" {
		t.Fatalf("Incorrect error: %v", err)
	}s
}

// type HTTPHandlerMock struct {
// 	mock.Mock
// }

// func (handler *HTTPHandlerMock) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
// 	handler.Called(wr, req)
// }

// func TestGeoIPBasic(t *testing.T) {
// 	mwCfg := mw.CreateConfig()
// 	mwCfg.DBPath = "./GeoLite2-City.mmdb"

// 	ctx := context.Background()
// 	next := new(HTTPHandlerMock)

// 	instance, err := mw.New(ctx, next, mwCfg, "traefik-geoip2")
// 	assert.NoError(t, err)

// 	recorder := httptest.NewRecorder()
// 	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

// 	next.On("ServeHTTP", mock.Anything, mock.Anything).Return()

// 	instance.ServeHTTP(recorder, req)
// 	assert.Equal(t, recorder.Result().StatusCode, 200)

// 	next.AssertCalled(t, "ServeHTTP", mock.Anything, mock.Anything)
// }

// func TestGeoIPFromRemoteAddr(t *testing.T) {
// 	mwCfg := mw.CreateConfig()
// 	mwCfg.DBPath = "./GeoLite2-City.mmdb"

// 	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
// 	instance, _ := mw.New(context.Background(), next, mwCfg, "traefik-geoip2")

// 	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
// 	req.RemoteAddr = "95.67.102.233"
// 	instance.ServeHTTP(httptest.NewRecorder(), req)
// 	assert.Equal(t, "UA", req.Header.Get(mw.CountryHeader))
// 	assert.Equal(t, "Kyiv City", req.Header.Get(mw.RegionHeader))
// 	assert.Equal(t, "Kyiv", req.Header.Get(mw.CityHeader))

// 	req = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
// 	req.RemoteAddr = "qwerty"
// 	instance.ServeHTTP(httptest.NewRecorder(), req)
// 	assert.Equal(t, mw.Unknown, req.Header.Get(mw.CountryHeader))
// 	assert.Equal(t, mw.Unknown, req.Header.Get(mw.RegionHeader))
// 	assert.Equal(t, mw.Unknown, req.Header.Get(mw.CityHeader))
// }

func TestGeoIPCountryDBFromRemoteAddr(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoLite2-Country.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.Background(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "95.67.102.233"
	instance.ServeHTTP(httptest.NewRecorder(), req)

	assertHeader(t, req, mw.CountryHeader, "UA")
	assertHeader(t, req, mw.RegionHeader, mw.Unknown)
	assertHeader(t, req, mw.CityHeader, mw.Unknown)
}

// func TestGeoIPFromXForwardedFrom(t *testing.T) {
// 	t.SkipNow()

// 	mwCfg := mw.CreateConfig()
// 	mwCfg.DBPath = "./GeoLite2-City.mmdb"

// 	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
// 	instance, _ := mw.New(context.Background(), next, mwCfg, "traefik-geoip2")

// 	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
// 	req.RemoteAddr = "1.1.1.1"
// 	req.Header.Set("X-Forwarded-For", "95.67.102.233")
// 	instance.ServeHTTP(httptest.NewRecorder(), req)
// 	assert.Equal(t, "UA", req.Header.Get(mw.CountryHeader))
// 	assert.Equal(t, "Kyiv City", req.Header.Get(mw.RegionHeader))
// 	assert.Equal(t, "Kyiv", req.Header.Get(mw.CityHeader))

// 	req = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
// 	req.RemoteAddr = "qwerty"
// 	instance.ServeHTTP(httptest.NewRecorder(), req)
// 	assert.Equal(t, "XX", req.Header.Get(mw.CountryHeader))
// 	assert.Equal(t, "XX", req.Header.Get(mw.RegionHeader))
// 	assert.Equal(t, "XX", req.Header.Get(mw.CityHeader))
// }

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if req.Header.Get(key) != expected {
		t.Fatalf("invalid header %s != %s", key, req.Header.Get(key))
	}
}
