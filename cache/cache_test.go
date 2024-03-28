package cache

import (
	"testing"
	"time"
)

type bbb struct {
	BBB string
}

type aaa struct {
	Name Cache[bbb]
}

func TestCache(t *testing.T) {
	a := &aaa{
		Name: NewAnyCache[bbb](time.Minute),
	}
	time.Sleep(time.Second * 3)
	a.Name.Close()
	time.Sleep(time.Second)
}
