package traefikgeoip2

import "net/http"

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
