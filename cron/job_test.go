package cron

import (
	"testing"
	"time"
)

func TestJob(t *testing.T) {
	a := NewCrontab()
	err := a.AddWithLimits("test1", "*/2  *   * * *    *", 3, func() {
		defer func() {
			recover()
		}()
		println(time.Now().String())
	})
	if err != nil {
		println(err.Error())
		return
	}
	for {
		time.Sleep(time.Second * 10)
		if !a.running {
			return
		}
	}
}

func TestJob2(t *testing.T) {
	a := NewCrontab()
	err := a.AddWithLimits("test1", "*/3  *   * * *    *", 3, func() {
		defer func() {
			recover()
		}()
		println(time.Now().String())
	})
	if err != nil {
		println(err.Error())
		return
	}
	for {
		time.Sleep(time.Second * 10)
		if !a.running {
			return
		}
	}
}
