package coord

import (
	"fmt"
	"strings"

	"github.com/xyzj/gopsu"
)

var georep = strings.NewReplacer("(", "", ")", "", ", ", ",", "POINT", "", "POLYGON", "", "LINESTRING", "", "POINT ", "", "POLYGON ", "", "LINESTRING ", "") // 经纬度字符串处理替换器

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

func Text2Geo(s string) []*Point {
	geostr := strings.Split(georep.Replace(s), ",")
	gp := make([]*Point, 0)
	for _, v := range geostr {
		vv := strings.Split(v, " ")
		gp = append(gp, &Point{
			Lng: gopsu.String2Float64(vv[0]),
			Lat: gopsu.String2Float64(vv[1]),
		})
	}
	return gp
}

func Geo2Text(gp []*Point) string {
	geostr := "POINT(0 0)" // 默认值，上海
	switch len(gp) {
	case 0: // 没有位置
	case 1: // 点
		geostr = fmt.Sprintf("POINT(%f %f)", gp[0].Lng, gp[0].Lat)
	default: // 线或者面
		pts := make([]string, len(gp))
		for k, v := range gp {
			pts[k] = fmt.Sprintf("%f %f", v.Lng, v.Lat)
		}
		if pts[0] == pts[len(gp)-1] { // 前后2点一致，表示面
			geostr = fmt.Sprintf("POLYGON((%s))", strings.Join(pts, ","))
		} else {
			geostr = fmt.Sprintf("LINESTRING(%s)", strings.Join(pts, ","))
		}
	}
	return geostr
}
