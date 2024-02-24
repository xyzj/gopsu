package cron

import (
	"strings"
	"testing"
	"time"
)

func TestJob2(t *testing.T) {
	t3 := time.Now().Add(time.Second * -5)
	time.Sleep(time.Second * 3)
	println(time.Until(t3).Seconds())
}
func TestJob1(t *testing.T) {
	a := NewCrontab()
	if a == nil {
		t.Fail()
		return
	}
	err := a.Add("test1", "* * * * *", func() {
		println("cron job " + time.Now().String())
	})
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	err = a.AddWithLimits("test2", 1, time.Now(), time.Second*15, func() {
		println("limit job " + time.Now().String())
	})
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	time.Sleep(time.Second * 2)
	a.Remove("test1")
	println(strings.Join(a.jobs.Keys(), ", "))

	time.Sleep(time.Minute * 5)
}
