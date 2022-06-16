package loopfunc

import (
	"sync"
	"time"
)

// LoopFunc 执行循环工作，并提供panic恢复
func LoopFunc(f func(params ...interface{}), params ...interface{}) {
	locker := &sync.WaitGroup{}
	end := false
RUN:
	locker.Add(1)
	go func() {
		defer func() {
			if err := recover(); err == nil {
				// 非panic,不需要恢复
				end = true
			}
			locker.Done()
		}()
		f(params...)
	}()
	locker.Wait()
	if end {
		return
	}
	time.Sleep(time.Second * 3)
	goto RUN
}
