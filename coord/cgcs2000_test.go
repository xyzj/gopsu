package coord

import "testing"

func TestCGCS2000(t *testing.T) {
	p1 := &Point{
		Lng: 17865.14,
		Lat: 35902.44,
	}
	// p2 := &Point{}
	east := 40500000.0
	meridian := 120.0
	t.Run("cgcs to wgs v1", func(t *testing.T) {
		// xp1 := WGS84ToCGCS2000(p1, east, meridian)
		// println(xp1.String())
		// xp2 := WGS84ToCGCS2000v2(p1, east, meridian)
		// println(xp2.String())
		yp1 := CGCS2000ToWGS84(p1, east, meridian)
		yp2 := CGCS2000ToWGS84v2(p1, east, meridian)
		println(yp1.String())
		println(yp2.String())
	})
}
