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
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/tovenja/cron/v3"
	"github.com/xyzj/gopsu/mapfx"
)

type job struct {
	job     func()
	name    string
	spec    string
	limits  uint
	running bool
}
type Crontab struct {
	parser  cron.Parser
	cron    gocron.Scheduler
	jobs    *mapfx.StructMap[string, job]
	running bool
}

// Add 添加一个循环任务
//
//	name： 任务名称，不可重复
//	spec： 执行间隔，crontab格式
//	do: 任务执行内容
func (c *Crontab) Add(name, spec string, do func()) error {
	if !c.running {
		return fmt.Errorf("scheduler is not ready")
	}

	if c.jobs.Has(name) {
		return fmt.Errorf("job " + name + " already exist")
	}

	if _, err := c.parser.Parse(spec); err != nil {
		if strings.HasPrefix(err.Error(), "expected exactly 6 fields, found 5") { // 采用随机秒
			spec = strconv.Itoa(rand.Intn(60)) + " " + spec
		}
	}
	_, err := c.cron.NewJob(
		gocron.CronJob(spec, true),
		gocron.NewTask(do),
		gocron.WithTags(name),
	)
	if err != nil {
		return err
	}
	c.jobs.Store(name, &job{
		spec:    spec,
		job:     do,
		name:    name,
		running: true,
	})
	return nil
}

// AddWithLimits 添加有限次数的任务，此类任务有时效性，因此无法暂停，只能删除
//
//	name： 任务名称，不可重复
//	limits: 执行次数
//	startAt 任务开始时间
//	dur: 任务执行间隔
//	do: 任务执行内容
func (c *Crontab) AddWithLimits(name string, limits uint, startAt time.Time, dur time.Duration, do func()) error {
	if !c.running {
		return fmt.Errorf("scheduler is not ready")
	}

	if c.jobs.Has(name) {
		return fmt.Errorf("job " + name + " already exist")
	}
	if limits == 0 {
		return fmt.Errorf("limits should be more than zero")
	}
	flimit := func(jobName string) {
		if j, ok := c.jobs.LoadForUpdate(jobName); ok {
			if j.limits > 0 {
				j.limits--
			}
			if j.limits == 0 {
				c.jobs.Delete(jobName)
			}
		}
	}
	opts := []gocron.JobOption{
		gocron.WithTags(name),
		gocron.WithName(name),
		gocron.WithLimitedRuns(limits),
		gocron.WithEventListeners(
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) { flimit(jobName) }),
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) { flimit(jobName) }),
		),
	}
	if time.Until(startAt).Seconds() > 1 {
		opts = append(opts, gocron.JobOption(gocron.WithStartDateTime(startAt)))
	} else {
		opts = append(opts, gocron.JobOption(gocron.WithStartImmediately()))
	}
	_, err := c.cron.NewJob(
		gocron.DurationJob(dur),
		gocron.NewTask(do),
		opts...,
	)
	if err != nil {
		return err
	}
	c.jobs.Store(name, &job{
		job:     do,
		name:    name,
		limits:  limits,
		running: true,
	})
	return nil
}

// Remove 删除指定任务
//
//	name： 任务名称
func (c *Crontab) Remove(name ...string) error {
	if !c.running {
		return fmt.Errorf("scheduler is not ready")
	}
	c.cron.RemoveByTags(name...)
	c.jobs.DeleteMore(name...)
	return nil
}

// Pause 暂停指定的循环任务，有限执行次数的任务无法暂停，只能删除
//
//	name： 任务名称
func (c *Crontab) Pause(name string) error {
	if !c.running {
		return fmt.Errorf("scheduler is not ready")
	}
	if j, ok := c.jobs.LoadForUpdate(name); ok {
		if j.spec == "" {
			return fmt.Errorf("limits job can not be pause, can only be remove")
		}
		if j.running {
			c.cron.RemoveByTags(name)
			j.running = false
		}
		return nil
	}
	return fmt.Errorf("job " + name + " does not exist")
}

// Resume 继续执行指定的循环任务
//
//	name： 任务名称
func (c *Crontab) Resume(name string) error {
	if !c.running {
		return fmt.Errorf("scheduler is not ready")
	}

	if j, ok := c.jobs.LoadForUpdate(name); ok {
		if j.running {
			return nil
		}
		if j.spec != "" {
			_, err := c.cron.NewJob(
				gocron.CronJob(j.spec, true),
				gocron.NewTask(j.job),
				gocron.WithTags(name),
			)
			if err != nil {
				return err
			}
			j.running = true
		}
		return nil
	}
	return fmt.Errorf("job " + name + " does not exist")
}

// Clean 清除所有任务
func (c *Crontab) Clean() {
	c.cron.RemoveByTags(c.jobs.Keys()...)
	c.jobs.Clean()
}

// List 列出所有任务名称
func (c *Crontab) List() []string {
	return c.jobs.Keys()
}

// NewCrontab 创建一个新的计划任务
func NewCrontab() *Crontab {
	// p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sc, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return &Crontab{
			running: false,
		}
	}
	sc.Start()
	return &Crontab{
		parser:  cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		cron:    sc,
		jobs:    mapfx.NewStructMap[string, job](),
		running: true,
	}
}
