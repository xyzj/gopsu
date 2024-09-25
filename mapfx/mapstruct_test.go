package mapfx

import (
	"fmt"
	"testing"
)

type aaa struct {
	name   string
	count  int
	status bool
}

func TestStruct(t *testing.T) {
	a := NewStructMap[string, aaa]()
	a.Store("test1", &aaa{
		name:   "sdkfhakfd",
		count:  3,
		status: false,
	})

	a.Store("test3", &aaa{
		name:   "sdkfhakfd",
		count:  3,
		status: false,
	})

	a.Store("tes45", &aaa{
		name:   "sdkfhakfd",
		count:  3,
		status: false,
	})

	aa, _ := a.LoadForUpdate("test1")
	aa.count = 7
	aa.status = true

	err := a.ForEach(func(key string, value *aaa) bool {
		println(key, fmt.Sprintf("%+v", value))

		return true
	})
	println(fmt.Sprintf("++%+v", err))
	c := aaa{
		name:   "d232",
		count:  32,
		status: true,
	}
	d := *new(aaa)
	copy([]aaa{d}, []aaa{c})
	println(fmt.Sprintf("%+v", d), &d, &c)
}
