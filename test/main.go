package main

import (
	"fmt"

	"github.com/xyzj/gopsu"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	lon := 121.4890497
	lat := 31.2252985
	// t := time.Unix(gopsu.Time2Stamp("2020-01-11 00:00:00"), 0)
	ss := &gopsu.SunrisesetParams{}
	ss.Latitude = lat
	ss.Longitude = lon
	// ss.UtcOffset = 8
	println(ss.Calculation())
	// ss.SunResult.Range(func(key, value interface{}) bool {
	// 	vv := value.(*gopsu.SunrisesetResult)
	// 	println(fmt.Sprintf("%02d-%02d --> %02d:%02d - %02d:%02d", vv.Month, vv.Day, vv.Sunrise/60, vv.Sunrise%60, vv.Sunset/60, vv.Sunset%60))
	// 	return true
	// })
	c, d := ss.Get(5, 11)
	println(fmt.Sprintf("%02d:%02d - %02d:%02d", c/60, c%60, d/60, d%60))
}
