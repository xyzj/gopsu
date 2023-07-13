package coord

import (
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	offset      = 0.00669342162296594323
	axis        = 6378245.0
	earthRadius = 6372797.560856
)

// IsOutOFChina 范围检测
func IsOutOFChina(p *Point) bool {
	lon, lat := p.Lng, p.Lat
	return !(lon > 72.004 && lon < 135.05 && lat > 3.86 && lat < 53.55)
}

// deltaP
func delta(p *Point) *Point {
	lon, lat := p.Lng, p.Lat
	tp := transform(&Point{Lng: lon - 105.0, Lat: lat - 35.0})
	dlat, dlon := tp.Lng, tp.Lat
	radlat := lat / 180.0 * math.Pi
	magic := math.Sin(radlat)
	magic = 1 - offset*magic*magic
	sqrtmagic := math.Sqrt(magic)

	dlat = (dlat * 180.0) / ((axis * (1 - offset)) / (magic * sqrtmagic) * math.Pi)
	dlon = (dlon * 180.0) / (axis / sqrtmagic * math.Cos(radlat) * math.Pi)

	return &Point{
		Lng: lon + dlon,
		Lat: lat + dlat,
	}
}

// transform location transform
func transform(p *Point) *Point {
	lon, lat := p.Lng, p.Lat
	var lonlat = lon * lat
	var absX = math.Sqrt(math.Abs(lon))
	var lonPi, latPi = lon * math.Pi, lat * math.Pi
	var d = 20.0*math.Sin(6.0*lonPi) + 20.0*math.Sin(2.0*lonPi)
	x, y := d, d
	x += 20.0*math.Sin(latPi) + 40.0*math.Sin(latPi/3.0)
	y += 20.0*math.Sin(lonPi) + 40.0*math.Sin(lonPi/3.0)
	x += 160.0*math.Sin(latPi/12.0) + 320*math.Sin(latPi/30.0)
	y += 150.0*math.Sin(lonPi/12.0) + 300.0*math.Sin(lonPi/30.0)
	x *= 2.0 / 3.0
	y *= 2.0 / 3.0
	x += -100.0 + 2.0*lon + 3.0*lat + 0.2*lat*lat + 0.1*lonlat + 0.2*absX
	y += 300.0 + lon + 2.0*lat + 0.1*lon*lon + 0.1*lonlat + 0.1*absX
	return &Point{Lng: x, Lat: y}
}

// WGS84toGCJ02 WGS84坐标系->火星坐标系
func WGS84toGCJ02(p *Point) *Point {
	if IsOutOFChina(p) {
		return p
	}
	return delta(p)
}

// GCJ02toWGS84 火星坐标系->WGS84坐标系
func GCJ02toWGS84(p *Point) *Point {
	if IsOutOFChina(p) {
		return p
	}
	m := delta(p)
	return &Point{
		Lng: p.Lng*2 - m.Lng,
		Lat: p.Lat*2 - m.Lat,
	}
}

// BD09toWGS84 百度坐标系->WGS84坐标系
func BD09toWGS84(p *Point) *Point {
	return GCJ02toWGS84(BD09toGCJ02(p))
}

// WGS84toBD09 WGS84坐标系->百度坐标系
func WGS84toBD09(p *Point) *Point {
	return GCJ02toBD09(WGS84toGCJ02(p))
}

// ToMercator 任意坐标系->墨卡托坐标系
func ToMercator(p *Point) *Point {
	x, y := p.Lng, p.Lat
	x = x * 20037508.34 / 180
	y = math.Log(math.Tan((90+y)*math.Pi/360)) / (math.Pi / 180)
	y = y * 20037508.34 / 180
	return &Point{Lng: x, Lat: y}
}

// FromMercator 墨卡托坐标系->任意坐标系
func FromMercator(p *Point) *Point {
	x, y := p.Lng, p.Lat
	x = x / 20037508.34 * 180
	y = y / 20037508.34 * 180
	y = 180 / math.Pi * (2*math.Atan(math.Exp(y*math.Pi/180)) - math.Pi/2)
	return &Point{Lng: x, Lat: y}
}

func degRad(ang float64) float64 {
	return ang * gc
}

// Distance computes the distance between two given coordinates in meter
func Distance(longitude1, latitude1, longitude2, latitude2 float64) float64 {
	radLat1 := degRad(latitude1)
	radLat2 := degRad(latitude2)
	a := radLat1 - radLat2
	b := degRad(longitude1) - degRad(longitude2)
	return 2 * earthRadius * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+
		math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2)))
}

// RandomGPS 生成半径内的随机经纬度
//
//	lon,lat: 中心点
//	radius：范围半径，单位米
//	返回：经度，纬度
func RandomGPS(lon, lat float64, radius float64) *Point {
	// rand.Seed(time.Now().UnixNano())
	radiusInDegrees := radius / 111300
	u := rand.Float64()
	v := rand.Float64()
	w := radiusInDegrees * math.Sqrt(u)
	t := math.Pi * v * 2
	x := w * math.Cos(t)
	y := w * math.Sin(t)
	return &Point{Lng: lon + y, Lat: lat + x}
}
