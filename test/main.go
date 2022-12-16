package main

import (
	"fmt"
	"math"
	"time"

	"github.com/xyzj/go-pinyin"
)

const (
	dr          = math.Pi / 180.0
	earthRadius = 6372797.560856
)

func aaa(a, b, c string, d, e int) {
	println(fmt.Sprintf("%s, %s, %s, --- %d %d", a, b, c, d, e))
}

type sliceFlag []string

func (f *sliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*f))
}

func (f *sliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var (
	dirs sliceFlag
)

// xCacheData 可设置超时的缓存字典数据结构
type xCacheData struct {
	Value  interface{}
	Expire time.Time
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
func GeoDistance2(lng1 float64, lat1 float64, lng2 float64, lat2 float64) float64 {
	const PI float64 = 3.141592653589793
	radlat1 := PI * lat1 / 180
	radlat2 := PI * lat2 / 180

	theta := lng1 - lng2
	radtheta := PI * theta / 180

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515
	dist = dist * 1.609344
	return dist * 1000
}
func GetDistance(lng1, lat1, lng2, lat2 float64) float64 {
	radius := earthRadius //6378137.0
	rad := math.Pi / 180.0
	lat1 = lat1 * rad
	lng1 = lng1 * rad
	lat2 = lat2 * rad
	lng2 = lng2 * rad
	theta := lng2 - lng1
	dist := math.Acos(math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(theta))
	return dist * radius
}
func main() {
	println(pinyin.XPinyinMatch("常和路308号", "ch"))
	println(pinyin.XPinyinMatch("常和路308号", "cl"))
	println(pinyin.XPinyinMatch("常和路308号", "308"))
	println(pinyin.XPinyinMatch("常和路308号", "changhe"))
	println(pinyin.XPinyinMatch("常和路308号", "常"))
	println(pinyin.XPinyinMatch("常和路308号", "lu3"))
}
