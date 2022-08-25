package loopfunc

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// LoopFunc 执行循环工作，并提供panic恢复
func LoopFunc(f func(params ...interface{}), name string, logWriter io.Writer, params ...interface{}) {
	locker := &sync.WaitGroup{}
	end := false
	if logWriter != nil {
		log.SetFlags(log.Ltime)
		log.SetOutput(logWriter)
	}
RUN:
	locker.Add(1)
	func() {
		defer func() {
			if err := recover(); err == nil {
				// 非panic,不需要恢复
				end = true
			} else {
				if reflect.TypeOf(err).String() == "error" {
					log.Println(fmt.Sprintf("[LOOP] %s crash: %v", name, errors.WithStack(err.(error))))
				} else {
					log.Println(fmt.Sprintf("[LOOP] %s crash: %v", name, err))
				}
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
