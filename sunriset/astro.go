package sunriset

import (
	"time"

	"github.com/starainrt/astro/sun"
)

var (
	// LocalCST 本地时区
	LocalCST = time.FixedZone("CST", 8*3600)
)

// LeapYear 判断是否闰年
func LeapYear(year int) bool {
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}

// CalcSuntimeAstro 使用starainrt/astro库计算日出日落 // year int, month time.Month, day int
func CalcSuntimeAstro(lat float64, lng float64, date time.Time) (*time.Time, *time.Time, error) {
	// 指定2020年1月1日8时8分8秒
	rise, err := sun.RiseTime(date, lng, lat, 0, true)
	if err != nil {
		return nil, nil, err
	}
	set, err := sun.DownTime(date, lng, lat, 0, true)
	if err != nil {
		return nil, nil, err
	}
	return &rise, &set, nil
}
