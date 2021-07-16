package traefikgeoip2

import (
	"fmt"
	"net"
	"net/http"

	"github.com/IncSW/geoip2"
)

// Unknown constant for undefined data.
const Unknown = "XX"

// DefaultDBPath default GeoIP2 database path.
const DefaultDBPath = "GeoLite2-Country.mmdb"

const (
	// RealIPHeader real ip header.
	RealIPHeader = "X-Real-IP"
	// CountryHeader country header name.
	CountryHeader = "X-GeoIP2-Country"
	// RegionHeader region header name.
	RegionHeader = "X-GeoIP2-Region"
	// CityHeader city header name.
	CityHeader = "X-GeoIP2-City"
)

// GeoIPResult GeoIPResult.
type GeoIPResult struct {
	country string
	region  string
	city    string
}

// ApplyGeoIPResult ApplyGeoIPResult.
func ApplyGeoIPResult(req *http.Request, res *GeoIPResult) {
	req.Header.Set(CountryHeader, res.country)
	req.Header.Set(RegionHeader, res.region)
	req.Header.Set(CityHeader, res.city)
}

// LookupGeoIP2 LookupGeoIP2.
type LookupGeoIP2 func(ip net.IP) (*GeoIPResult, error)

// CreateCityDBLookup CreateCityDBLookup.
func CreateCityDBLookup(rdr *geoip2.CityReader) LookupGeoIP2 {
	return func(ip net.IP) (*GeoIPResult, error) {
		rec, err := rdr.Lookup(ip)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		retval := GeoIPResult{
			country: rec.Country.ISOCode,
			region:  Unknown,
			city:    rec.City.Names["en"],
		}
		if rec.Subdivisions != nil {
			retval.region = rec.Subdivisions[0].Names["en"]
		}
		return &retval, nil
	}
}

// CreateCountryDBLookup CreateCountryDBLookup.
func CreateCountryDBLookup(rdr *geoip2.CountryReader) LookupGeoIP2 {
	return func(ip net.IP) (*GeoIPResult, error) {
		rec, err := rdr.Lookup(ip)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		retval := GeoIPResult{
			country: rec.Country.ISOCode,
			region:  Unknown,
			city:    Unknown,
		}
		return &retval, nil
	}
}
