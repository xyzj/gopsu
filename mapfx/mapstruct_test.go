package mapfx

import "testing"

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

	aa, _ := a.LoadForUpdate("test1")
	aa.Count = 7
	aa.Status = true

	bb, _ := a.Load("test1")
	println(bb.Count, bb.Status)
}
