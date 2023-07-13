package coord

import (
	"fmt"
)

// Point point struct
type Point struct {
	Lng float64 `json:"lng"`
	Lat float64 `json:"lat"`
}

// String return lng, lat
func (p *Point) String() string {
	return fmt.Sprintf("%.12f %.12f", p.Lng, p.Lat)
}

// GeoText return mysql geotext
func (p *Point) GeoText() string {
	return fmt.Sprintf("POINT (%.12f %.12f)", p.Lng, p.Lat)
}

// Value return the lon and lat value
func (p *Point) Value() (float64, float64) {
	return p.Lng, p.Lat
}

// Equals this point is equivalent to the other point
func (p *Point) Equals(other *Point) bool {
	return p.Lng == other.Lng && p.Lat == other.Lat
}

// Round round this point to the nearest
func (p *Point) Round(l int) *Point {
	var a float64 = 1
	for i := 0; i < l; i++ {
		a *= 10
	}
	return &Point{
		Lng: float64(int(p.Lng*a+0.5)) / a,
		Lat: float64(int(p.Lat*a+0.5)) / a,
	}
}

// RoundString limit number after dot
func (p *Point) RoundString(l int) string {
	if l < 0 {
		l = 12
	}

	s := fmt.Sprintf("%%.%df %%.%df", l, l)
	return fmt.Sprintf(s, p.Lng, p.Lat)
}
