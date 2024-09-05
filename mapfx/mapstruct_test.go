package mapfx

import (
	"fmt"
	"testing"
)

type aaa struct {
	Name   string
	Count  int
	Status bool
}

func TestStruct(t *testing.T) {
	a := NewStructMap[string, aaa]()
	a.Store("test1", &aaa{
		Name:   "sdkfhakfd",
		Count:  3,
		Status: false,
	})

	a.Store("test3", &aaa{
		Name:   "sdkfhakfd",
		Count:  3,
		Status: false,
	})

	a.Store("tes45", &aaa{
		Name:   "sdkfhakfd",
		Count:  3,
		Status: false,
	})

	aa, _ := a.LoadForUpdate("test1")
	aa.Count = 7
	aa.Status = true

	err := a.ForEach(func(key string, value *aaa) bool {
		println(key)
		if key == "test3" {
			// panic(fmt.Errorf("panicd"))
		}
		return true
	})
	println(fmt.Sprintf("++%+v", err))
}
