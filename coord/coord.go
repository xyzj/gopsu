package coord

import (
	"math"
	"math/rand"
	"time"
)

// WGS84坐标系：即地球坐标系，国际上通用的坐标系。
// GCJ02坐标系：即火星坐标系，WGS84坐标系经加密后的坐标系。Google Maps，高德在用。
// BD09坐标系：即百度坐标系，GCJ02坐标系经加密后的坐标系。

const (
	// XPI XPI
	XPI = math.Pi * 3000.0 / 180.0
	// OFFSET OFFSET
	OFFSET = 0.00669342162296594323
	// AXIS AXIS
	AXIS        = 6378245.0
	dr          = math.Pi / 180.0
	earthRadius = 6372797.560856
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// BD09toGCJ02 百度坐标系->火星坐标系
func BD09toGCJ02(lon, lat float64) (float64, float64) {
	x := lon - 0.0065
	y := lat - 0.006

	z := math.Sqrt(x*x+y*y) - 0.00002*math.Sin(y*XPI)
	theta := math.Atan2(y, x) - 0.000003*math.Cos(x*XPI)

	gLon := z * math.Cos(theta)
	gLat := z * math.Sin(theta)

	return gLon, gLat
}

// GCJ02toBD09 火星坐标系->百度坐标系
func GCJ02toBD09(lon, lat float64) (float64, float64) {
	z := math.Sqrt(lon*lon+lat*lat) + 0.00002*math.Sin(lat*XPI)
	theta := math.Atan2(lat, lon) + 0.000003*math.Cos(lon*XPI)

	bdLon := z*math.Cos(theta) + 0.0065
	bdLat := z*math.Sin(theta) + 0.006

	return bdLon, bdLat
}

// WGS84toGCJ02 WGS84坐标系->火星坐标系
func WGS84toGCJ02(lon, lat float64) (float64, float64) {
	if isOutOFChina(lon, lat) {
		return lon, lat
	}

	mgLon, mgLat := delta(lon, lat)

	return mgLon, mgLat
}

// GCJ02toWGS84 火星坐标系->WGS84坐标系
func GCJ02toWGS84(lon, lat float64) (float64, float64) {
	if isOutOFChina(lon, lat) {
		return lon, lat
	}

	mgLon, mgLat := delta(lon, lat)

	return lon*2 - mgLon, lat*2 - mgLat
}

// BD09toWGS84 百度坐标系->WGS84坐标系
func BD09toWGS84(lon, lat float64) (float64, float64) {
	lon, lat = BD09toGCJ02(lon, lat)
	return GCJ02toWGS84(lon, lat)
}

// WGS84toBD09 WGS84坐标系->百度坐标系
func WGS84toBD09(lon, lat float64) (float64, float64) {
	lon, lat = WGS84toGCJ02(lon, lat)
	return GCJ02toBD09(lon, lat)
}

func delta(lon, lat float64) (float64, float64) {
	dlat, dlon := coordTransform(lon-105.0, lat-35.0)
	radlat := lat / 180.0 * math.Pi
	magic := math.Sin(radlat)
	magic = 1 - OFFSET*magic*magic
	sqrtmagic := math.Sqrt(magic)

	dlat = (dlat * 180.0) / ((AXIS * (1 - OFFSET)) / (magic * sqrtmagic) * math.Pi)
	dlon = (dlon * 180.0) / (AXIS / sqrtmagic * math.Cos(radlat) * math.Pi)

	mgLat := lat + dlat
	mgLon := lon + dlon

	return mgLon, mgLat
}
func coordTransform(lon, lat float64) (x, y float64) {
	var lonlat = lon * lat
	var absX = math.Sqrt(math.Abs(lon))
	var lonPi, latPi = lon * math.Pi, lat * math.Pi
	var d = 20.0*math.Sin(6.0*lonPi) + 20.0*math.Sin(2.0*lonPi)
	x, y = d, d
	x += 20.0*math.Sin(latPi) + 40.0*math.Sin(latPi/3.0)
	y += 20.0*math.Sin(lonPi) + 40.0*math.Sin(lonPi/3.0)
	x += 160.0*math.Sin(latPi/12.0) + 320*math.Sin(latPi/30.0)
	y += 150.0*math.Sin(lonPi/12.0) + 300.0*math.Sin(lonPi/30.0)
	x *= 2.0 / 3.0
	y *= 2.0 / 3.0
	x += -100.0 + 2.0*lon + 3.0*lat + 0.2*lat*lat + 0.1*lonlat + 0.2*absX
	y += 300.0 + lon + 2.0*lat + 0.1*lon*lon + 0.1*lonlat + 0.1*absX
	return
}

func isOutOFChina(lon, lat float64) bool {
	return !(lon > 72.004 && lon < 135.05 && lat > 3.86 && lat < 53.55)
}

func degRad(ang float64) float64 {
	return ang * dr
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
func RandomGPS(lon, lat float64, radius float64) (float64, float64) {
	// rand.Seed(time.Now().UnixNano())
	radiusInDegrees := radius / 111300
	u := rand.Float64()
	v := rand.Float64()
	w := radiusInDegrees * math.Sqrt(u)
	t := math.Pi * v * 2
	x := w * math.Cos(t)
	y := w * math.Sin(t)
	return lon + y, lat + x
}
