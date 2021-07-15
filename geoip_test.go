package traefikgeoip2_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mw "github.com/GiGInnovationLabs/traefik-geoip2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGeoIPConfig(t *testing.T) {
	mwCfg := mw.CreateConfig()
	assert.Equal(t, mw.DefaultDBPath, mwCfg.DBPath)

	_, err := mw.New(context.TODO(), nil, mwCfg, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mw.DefaultDBPath)

	mwCfg.DBPath = "Makefile"
	_, err = mw.New(context.TODO(), nil, mwCfg, "")
	assert.EqualError(t, err, "error opening database: invalid MaxMind DB file")
}

type HTTPHandlerMock struct {
	mock.Mock
}

func (handler *HTTPHandlerMock) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	handler.Called(wr, req)
}

func TestGeoIPBasic(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoIP2-City.mmdb"

	ctx := context.Background()
	next := new(HTTPHandlerMock)

	instance, err := mw.New(ctx, next, mwCfg, "traefik-geoip2")
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

	next.On("ServeHTTP", mock.Anything, mock.Anything).Return()

	instance.ServeHTTP(recorder, req)
	assert.Equal(t, recorder.Result().StatusCode, 200)

	next.AssertCalled(t, "ServeHTTP", mock.Anything, mock.Anything)
}

func TestGeoIPFromRemoteAddr(t *testing.T) {
	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoIP2-City.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.Background(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "95.67.102.233"
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "UA", req.Header.Get(mw.CountryHeader))
	assert.Equal(t, "Kyiv City", req.Header.Get(mw.RegionHeader))
	assert.Equal(t, "Kyiv", req.Header.Get(mw.CityHeader))

	req = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "qwerty"
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "XX", req.Header.Get(mw.CountryHeader))
	assert.Equal(t, "XX", req.Header.Get(mw.RegionHeader))
	assert.Equal(t, "XX", req.Header.Get(mw.CityHeader))
}

func TestGeoIPFromXForwardedFrom(t *testing.T) {
	t.SkipNow()

	mwCfg := mw.CreateConfig()
	mwCfg.DBPath = "./GeoIP2-City.mmdb"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	instance, _ := mw.New(context.Background(), next, mwCfg, "traefik-geoip2")

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "1.1.1.1"
	req.Header.Set("X-Forwarded-For", "95.67.102.233")
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "UA", req.Header.Get(mw.CountryHeader))
	assert.Equal(t, "Kyiv City", req.Header.Get(mw.RegionHeader))
	assert.Equal(t, "Kyiv", req.Header.Get(mw.CityHeader))

	req = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	req.RemoteAddr = "qwerty"
	instance.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "XX", req.Header.Get(mw.CountryHeader))
	assert.Equal(t, "XX", req.Header.Get(mw.RegionHeader))
	assert.Equal(t, "XX", req.Header.Get(mw.CityHeader))
}
