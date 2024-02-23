// Package cron scheduled jobs
//
// 定时任务的执行精度为秒，秒的设置为可选，不指定秒时，模块会为你随机设置一个
// ┌──────────── second (0-59) 非必填
// │ ┌───────────── minute (0–59) 必填
// │ │ ┌───────────── hour (0–23) 必填
// │ │ │ ┌───────────── day of the month (1–31) 必填，不建议设置大于28，避免2月不执行
// │ │ │ │ ┌───────────── month (1–12) 必填
// │ │ │ │ │ ┌───────────── day of the week (0–6) (Sunday to Saturday) 必填
// * * * * * *
//
// 可用的特殊字符
// `*`: 表示每一秒/分钟/小时/天/月/周
// `,`: 分割指定的时间，如：1,2
// `-`: 指定一个区段，如：4-19
// `/`: 设置一个指定的周期，一般需要搭配`*`使用，如：*/20
package cron

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/tovenja/cron/v3"
	"github.com/xyzj/gopsu/mapfx"
)

type job struct {
	Job    func()
	Name   string
	Spec   string
	Detail string
	jobID  cron.EntryID
	times  int
}
type Crontab struct {
	parser  cron.Parser
	cron    *cron.Cron
	jobs    *mapfx.StructMap[string, job]
	running bool
}

func (c *Crontab) add(name, spec, detail string, times int, do func()) error {
	spec = strings.TrimSpace(spec)
	if name == "" || spec == "" || do == nil {
		return fmt.Errorf("more job information are needed")
	}
	if _, err := c.parser.Parse(spec); err != nil {
		if strings.HasPrefix(err.Error(), "expected exactly 6 fields, found 5") { // 采用随机秒
			spec = strconv.Itoa(rand.Intn(60)) + " " + spec
		} else {
			return err
		}
	}
	jid, err := c.cron.AddFunc(spec, do)
	if err != nil {
		return err
	}

	c.jobs.Store(name, &job{
		Name:   name,
		Spec:   spec,
		Detail: detail,
		Job:    do,
		jobID:  jid,
		times:  times,
	})
	c.start()
	return nil
}
func (c *Crontab) stop() {
	if c.jobs.Len() == 0 {
		c.running = false
		c.cron.Stop()
	}
}

func (c *Crontab) start() {
	if !c.running {
		c.running = true
		c.cron.Start()
	}
}

// countJobTimes 倒数任务执行次数，当执行次数为0时，删除任务
func (c *Crontab) countJobTimes(name string) {
	if j, ok := c.jobs.LoadForUpdate(name); ok {
		if j.times == -1 {
			return
		}
		if j.times > 0 {
			j.times--
		}
		if j.times == 0 {
			c.Remove(name)
		}
	}
}

// Add 添加任务
//
//	name： 任务名称，不可重复
//	spec： 执行间隔，crontab格式
//	detail： 任务说明，非必要
//	do: 任务执行内容
func (c *Crontab) Add(name, spec string, do func()) error {
	if c.jobs.Has(name) {
		return fmt.Errorf("job already exist")
	}

	return c.add(name, spec, "", -1, do)
}

// AddWithLimits 添加有限次数的任务，任务次数为-1时表示无限次数执行
//
//	name： 任务名称，不可重复
//	spec： 执行间隔，crontab格式
//	detail： 任务说明，非必要
//	times： 任务执行次数，-1表示无限次执行
//	do: 任务执行内容
func (c *Crontab) AddWithLimits(name, spec string, times int, do func()) error {
	if times == 0 {
		return fmt.Errorf("times should be -1 or more than zero")
	}
	if times < 0 {
		return c.Add(name, spec, do)
	}

	return c.add(name, spec, "", times, func() {
		defer func() {
			recover()
			c.countJobTimes(name)
		}()
		do()
	})
}

// Remove 删除指定任务
//
//	name： 任务名称
func (c *Crontab) Remove(name string) {
	if j, ok := c.jobs.Load(name); ok {
		c.cron.Remove(j.jobID)
		c.jobs.Delete(name)
	}
	c.stop()
}

// Clean 清除所有任务
func (c *Crontab) Clean() {
	c.jobs.ForEach(func(key string, value *job) bool {
		c.cron.Remove(value.jobID)
		return true
	})
	c.jobs.Clean()
	c.stop()
}

// Pause 暂停指定任务
//
//	name： 任务名称
func (c *Crontab) Pause(name string) error {
	if j, ok := c.jobs.LoadForUpdate(name); ok {
		c.cron.Remove(j.jobID)
		j.jobID = 0
		return nil
	}
	return fmt.Errorf("job not exist")
}

// Resume 继续执行指定任务
//
//	name： 任务名称
func (c *Crontab) Resume(name string) error {
	if j, ok := c.jobs.LoadForUpdate(name); ok {
		if j.jobID != 0 { // 已经运行
			return nil
		}
		jid, err := c.cron.AddFunc(j.Spec, j.Job)
		if err != nil {
			return err
		}
		j.jobID = jid
		c.start()
		return nil
	}
	return fmt.Errorf("job not exist")
}

// List 列出所有任务名称
func (c *Crontab) List() []string {
	return c.jobs.Keys()
}

// NewCrontab 创建一个新的计划任务
func NewCrontab() *Crontab {
	p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return &Crontab{
		parser: p,
		cron:   cron.New(cron.WithParser(p)),
		jobs:   mapfx.NewStructMap[string, job](),
	}
}
