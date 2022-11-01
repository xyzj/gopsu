package main

import (
	"fmt"
	"time"

	"github.com/xyzj/gopsu/sunriseset"
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

func main() {
	var tNow = time.Now()
	var tStart = time.Date(tNow.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	var sunTime = make(map[string][]int)
	// var locker = sync.WaitGroup{}
	var maxdays = 365

	if sunriseset.LeapYear(tNow.Year()) {
		maxdays = 366
	}
	for i := 0; i < maxdays; i++ {
		tCalc := tStart.AddDate(0, 0, i)
		rise, set, err := sunriseset.CalcSuntimeAstro(31.2465, 121.4914, tCalc)
		if err == nil {
			sunTime[fmt.Sprintf("%02d%02d", rise.Month(), rise.Day())] = []int{rise.Hour()*60 + rise.Minute(), set.Hour()*60 + set.Minute()}
		}
	}
	if _, ok := sunTime["0229"]; !ok {
		sunTime["0229"] = sunTime["0228"]
	}
	for k, v := range sunTime {
		println(k, fmt.Sprintf("%+v", v))
	}
	println(len(sunTime))
}
