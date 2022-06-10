package loopfunc

import (
	"sync"
	"time"
)

// LoopFunc 执行循环工作，并提供panic恢复
func LoopFunc(f func(params ...interface{}), params ...interface{}) {
	locker := &sync.WaitGroup{}
RUN:
	locker.Add(1)
	go func() {
		defer func() {
			recover()
			locker.Done()
		}()
		f(params...)
	}()
	locker.Wait()
	time.Sleep(time.Second * 3)
	goto RUN
}
