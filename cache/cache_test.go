package cache

import (
	"strconv"
	"testing"
	"time"
)

type bbb struct {
	BBB string
}

type aaa struct {
	Name Cache[bbb]
}

func BenchmarkCache(t *testing.B) {
	a := NewAnyCache[*bbb](time.Hour)
	t.ResetTimer()
	for i := 0; i < 1000000; i++ {
		a.Store(strconv.Itoa(i+1), &bbb{BBB: "string"})
	}
	// for i := 0; i < 1000000; i++ {
	// 	a.Load(strconv.Itoa(i + 1))
	// }
}
