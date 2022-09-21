/*
Package loopfunc ： 用于控制需要持续运行的循环方法，当方法漰溃时会自动重启
*/
package loopfunc

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// LoopFunc 执行循环工作，并提供panic恢复
//
// f: 要执行的循环方法，可控制传入参数
//
// name：这个方法的名称，用于错误标识
//
// logWriter：方法崩溃时的日志记录器，默认os.stdout
//
// params： 需要传给f的参数，f内需要进行类型转换
func LoopFunc(f func(params ...interface{}), name string, logWriter io.Writer, params ...interface{}) {
	locker := &sync.WaitGroup{}
	end := false
	if logWriter == nil {
		logWriter = os.Stdout
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
					logWriter.Write([]byte(fmt.Sprintf("%s [LOOP] crash: %v\n", name, errors.WithStack(err.(error)))))
				} else {
					logWriter.Write([]byte(fmt.Sprintf("%s [LOOP] crash: %v\n", name, err)))
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

// GoFunc 执行安全的子线程工作，包含panic捕获
//
// f: 要执行的循环方法，可控制传入参数
//
// name：这个方法的名称，用于错误标识
//
// logWriter：方法崩溃时的日志记录器，默认os.stdout
//
// params： 需要传给f的参数，f内需要进行类型转换
func GoFunc(f func(params ...interface{}), name string, logWriter io.Writer, params ...interface{}) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if reflect.TypeOf(err).String() == "error" {
					logWriter.Write([]byte(fmt.Sprintf("%s [GO] crash: %v\n", name, errors.WithStack(err.(error)))))
				} else {
					logWriter.Write([]byte(fmt.Sprintf("%s [GO] crash: %v\n", name, err)))
				}
			}
		}()
		f(params...)
	}()
}
