// Package proc watch process status
package proc

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/mohae/deepcopy"
	"github.com/shirou/gopsutil/process"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/cache"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
)

//go:embed echarts.min.js
var echarts []byte

type procStatus struct {
	dt     string
	Memrss uint64
	Memvms uint64
	GNum   int
	TNum   int
	Cpup   float32
	Memp   float32
}

func (ps *procStatus) String() string {
	return fmt.Sprintf("cpu: %.2f%%; mem: %.2f%%; rss: %s", ps.Cpup, ps.Memp, gopsu.FormatFileSize(ps.Memrss))
}

func (ps *procStatus) HTML() string {
	return fmt.Sprintf("cpu: %.2f%%\nmem: %.2f%%\nrss: %s", ps.Cpup, ps.Memp, gopsu.FormatFileSize(ps.Memrss))
}

func (ps *procStatus) JSON() string {
	js, _ := sjson.Set("", "cpu", fmt.Sprintf("%.2f", ps.Cpup))
	js, _ = sjson.Set(js, "mem", fmt.Sprintf("%.2f", ps.Memp))
	js, _ = sjson.Set(js, "rss", fmt.Sprintf("%d", ps.Memrss))
	js, _ = sjson.Set(js, "vms", fmt.Sprintf("%d", ps.Memvms))
	return js
}

type RecordOpt struct {
	Logg        logger.Logger
	Timer       time.Duration
	DataTimeout time.Duration
	Name        string
}
type Recorder struct {
	lastProc  *procStatus
	procCache *cache.AnyCache[*procStatus]
	opt       *RecordOpt
}

func (r *Recorder) LastHTML() string {
	return r.lastProc.HTML()
}

func (r *Recorder) LastJSON() string {
	return r.lastProc.JSON()
}

func (r *Recorder) LastString() string {
	return r.lastProc.String()
}

// StartRecord 记录进程状态
func StartRecord(opt *RecordOpt) *Recorder {
	if opt == nil {
		opt = &RecordOpt{}
	}
	if opt.Logg == nil {
		opt.Logg = &logger.NilLogger{}
	}
	if opt.Timer < time.Second*5 {
		opt.Timer = time.Second * 60
	}
	if opt.DataTimeout < time.Minute {
		opt.DataTimeout = time.Minute
	}
	if opt.DataTimeout > time.Hour*24*366 {
		opt.DataTimeout = time.Hour * 24 * 366
	}
	r := &Recorder{
		opt:       opt,
		lastProc:  &procStatus{},
		procCache: cache.NewAnyCache[*procStatus](opt.DataTimeout),
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		var proc *process.Process
		var err error
		var memi *process.MemoryInfoStat
		f := func() {
			if proc == nil {
				proc, err = process.NewProcess(int32(os.Getpid()))
				if err != nil {
					opt.Logg.Error("[PROC] " + err.Error())
					return
				}
			}
			v, _ := proc.CPUPercent()
			r.lastProc.Cpup = float32(v)
			r.lastProc.Memp, _ = proc.MemoryPercent()
			memi, _ = proc.MemoryInfo()
			if memi == nil {
				memi = &process.MemoryInfoStat{}
			}
			r.lastProc.Memrss = memi.RSS
			r.lastProc.Memvms = memi.VMS
			r.procCache.Store(time.Now().Format("01-02 15:04:05"), deepcopy.Copy(r.lastProc).(*procStatus))
		}
		t := time.NewTicker(r.opt.Timer)
		c := 0
		for range t.C {
			f()
			c++
			if c%30 == 0 {
				c = 0
				opt.Logg.Info("[PROC] " + r.lastProc.String())
			}
		}
	}, "proc", opt.Logg.DefaultWriter())
	return r
}

func (r *Recorder) GinHandler(c *gin.Context) {
	js := make([]*procStatus, 0, r.procCache.Len())
	r.procCache.ForEach(func(key string, value *procStatus) bool {
		value.dt = key
		js = append(js, value)
		return true
	})
	sort.Slice(js, func(i, j int) bool {
		return js[i].dt < js[j].dt
	})
	switch c.Request.Method {
	case "POST":
		c.Set("data", js)
		c.Set("status", 1)
		c.JSON(200, c.Keys)
	case "GET":
		var nametail string
		if r.opt.Name != "" {
			nametail = " (" + r.opt.Name + ")"
		}
		l := len(js)
		x := make([]string, 0, l)
		width := c.Param("width")
		cpu := make([]opts.LineData, 0, l)
		mem := make([]opts.LineData, 0, l)
		rss := make([]opts.LineData, 0, l)
		vms := make([]opts.LineData, 0, l)
		for _, v := range js {
			x = append(x, v.dt)
			cpu = append(cpu, opts.LineData{Name: fmt.Sprintf("%.2f%%", v.Cpup), Value: v.Cpup, Symbol: "circle"})
			mem = append(mem, opts.LineData{Name: fmt.Sprintf("%.2f%%", v.Memp), Value: v.Memp, Symbol: "circle"})
			rss = append(rss, opts.LineData{Name: gopsu.FormatFileSize(v.Memrss), Value: v.Memrss / 1024 / 1024, Symbol: "circle"})
			vms = append(vms, opts.LineData{Name: gopsu.FormatFileSize(v.Memvms), Value: v.Memvms / 1024 / 1024, Symbol: "circle"})
		}
		line1 := charts.NewLine()
		line1.SetGlobalOptions(SetupLineGOpts(&LineOpt{
			PageTitle:   "Process Records" + nametail,
			Name:        "CPU & MEM Use" + nametail,
			Total:       float32(l),
			TTFormatter: "{a0}: <b>{b0}</b><br>{a1}: <b>{b1}</b>",
			YFormatter:  "{value} %",
			Width:       width,
		})...)
		line1.SetXAxis(x)
		line1.AddSeries("cpu", cpu).SetSeriesOptions(SetupLineSOpts()...)
		line1.AddSeries("mem", mem).SetSeriesOptions(SetupLineSOpts()...)

		lineRss := charts.NewLine()
		lineRss.SetGlobalOptions(SetupLineGOpts(&LineOpt{
			Name:        "Resident Set Size" + nametail,
			Total:       float32(l),
			TTFormatter: "<b>{b}</b>",
			YFormatter:  "{value} MB",
			Width:       width,
		})...)
		lineRss.SetXAxis(x).
			AddSeries("rss", rss).
			SetSeriesOptions(SetupLineSOpts(charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}))...)

		lineVms := charts.NewLine()
		lineVms.SetGlobalOptions(SetupLineGOpts(&LineOpt{
			Name:        "Virtual Memory Size" + nametail,
			Total:       float32(l),
			TTFormatter: "<b>{b}</b>",
			YFormatter:  "{value} MB",
			Width:       width,
		})...)
		lineVms.SetXAxis(x).
			AddSeries("vms", vms).
			SetSeriesOptions(SetupLineSOpts(charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}))...)
		a := components.NewPage()
		a.PageTitle = "Process Records" + nametail
		a.AddCharts(line1, lineRss, lineVms)
		b := &bytes.Buffer{}
		a.Render(b)
		c.Writer.Write(LocalEchartsJS(b.Bytes()))
	}
}

func (r *Recorder) HTTPHandler(w http.ResponseWriter, req *http.Request) {
	js := make([]*procStatus, 0, r.procCache.Len())
	r.procCache.ForEach(func(key string, value *procStatus) bool {
		value.dt = key
		js = append(js, value)
		return true
	})
	sort.Slice(js, func(i, j int) bool {
		return js[i].dt < js[j].dt
	})
	switch req.Method {
	case "POST":
		s, _ := sjson.SetBytes([]byte{}, "status", 1)
		s, _ = sjson.SetBytes(s, "data", js)
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(s)
	case "GET":
		values := req.URL.Query()
		var nametail string
		if r.opt.Name != "" {
			nametail = " (" + r.opt.Name + ")"
		}
		l := len(js)
		x := make([]string, 0, l)
		width := values.Get("width")
		cpu := make([]opts.LineData, 0, l)
		mem := make([]opts.LineData, 0, l)
		rss := make([]opts.LineData, 0, l)
		vms := make([]opts.LineData, 0, l)
		for _, v := range js {
			x = append(x, v.dt)
			cpu = append(cpu, opts.LineData{Name: fmt.Sprintf("%.2f%%", v.Cpup), Value: v.Cpup, Symbol: "circle"})
			mem = append(mem, opts.LineData{Name: fmt.Sprintf("%.2f%%", v.Memp), Value: v.Memp, Symbol: "circle"})
			rss = append(rss, opts.LineData{Name: gopsu.FormatFileSize(v.Memrss), Value: v.Memrss / 1024 / 1024, Symbol: "circle"})
			vms = append(vms, opts.LineData{Name: gopsu.FormatFileSize(v.Memvms), Value: v.Memvms / 1024 / 1024, Symbol: "circle"})
		}
		line1 := charts.NewLine()
		line1.SetGlobalOptions(SetupLineGOpts(&LineOpt{
			PageTitle:   "Process Records" + nametail,
			Name:        "CPU & MEM Use" + nametail,
			Total:       float32(l),
			TTFormatter: "{a0}: <b>{b0}</b><br>{a1}: <b>{b1}</b>",
			YFormatter:  "{value} %",
			Width:       width,
		})...)
		line1.SetXAxis(x)
		line1.AddSeries("cpu", cpu).SetSeriesOptions(SetupLineSOpts()...)
		line1.AddSeries("mem", mem).SetSeriesOptions(SetupLineSOpts()...)

		lineRss := charts.NewLine()
		lineRss.SetGlobalOptions(SetupLineGOpts(&LineOpt{
			Name:        "Resident Set Size" + nametail,
			Total:       float32(l),
			TTFormatter: "<b>{b}</b>",
			YFormatter:  "{value} MB",
			Width:       width,
		})...)
		lineRss.SetXAxis(x).
			AddSeries("rss", rss).
			SetSeriesOptions(SetupLineSOpts(charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}))...)

		lineVms := charts.NewLine()
		lineVms.SetGlobalOptions(SetupLineGOpts(&LineOpt{
			Name:        "Virtual Memory Size" + nametail,
			Total:       float32(l),
			TTFormatter: "<b>{b}</b>",
			YFormatter:  "{value} MB",
			Width:       width,
		})...)
		lineVms.SetXAxis(x).
			AddSeries("vms", vms).
			SetSeriesOptions(SetupLineSOpts(charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}))...)

		a := components.NewPage()
		a.PageTitle = "Process Records" + nametail
		a.AddCharts(line1, lineRss, lineVms)
		b := &bytes.Buffer{}
		a.Render(b)
		w.Write(LocalEchartsJS(b.Bytes()))
	}
}

type LineOpt struct {
	PageTitle   string
	Name        string
	TTFormatter string
	YFormatter  string
	Width       string
	Heigh       string
	Total       float32
	PageCount   float32
}

func SetupLineGOpts(lopt *LineOpt) []charts.GlobalOpts {
	if lopt == nil {
		lopt = &LineOpt{
			Name: "unknow",
		}
	}
	if lopt.Width == "" {
		lopt.Width = "2000px"
	}
	if lopt.Heigh == "0" {
		lopt.Heigh = "500px"
	}
	start := float32(0)
	if lopt.PageCount > 0 && lopt.Total > lopt.PageCount {
		start = (lopt.Total - lopt.PageCount) / lopt.Total * 100
	}
	return []charts.GlobalOpts{
		charts.WithAnimation(),
		charts.WithTitleOpts(opts.Title{
			Title: lopt.Name,
			Left:  "10%",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Show:      true,
			Formatter: lopt.TTFormatter,
			AxisPointer: &opts.AxisPointer{
				Show: true,
				Type: "line",
				Label: &opts.Label{
					Show: true,
				},
			},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: lopt.PageTitle,
			// Theme:     types.ChartThemeRiver,
			Width:  lopt.Width,
			Height: lopt.Heigh,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				Rotate:       30,
				ShowMinLabel: true,
				ShowMaxLabel: true,
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			AxisLabel: &opts.AxisLabel{
				Show:         true,
				ShowMinLabel: true,
				ShowMaxLabel: true,
				Formatter:    lopt.YFormatter,
			},
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Start: start,
		}),
	}
}

func SetupLineSOpts(lopt ...charts.SeriesOpts) []charts.SeriesOpts {
	serOpt := []charts.SeriesOpts{
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: true,
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Width: 2,
		}),
	}
	serOpt = append(serOpt, lopt...)
	return serOpt
}

func SetupBarSOpts(lopt ...charts.SeriesOpts) []charts.SeriesOpts {
	serOpt := []charts.SeriesOpts{
		charts.WithBarChartOpts(opts.BarChart{
			RoundCap: true,
		}),
	}
	serOpt = append(serOpt, lopt...)
	return serOpt
}

func EchartsJS() []byte {
	return json.Bytes("<script type=\"text/javascript\">" + json.String(echarts) + "\n</script>")
}

func LocalEchartsJS(htmlpage []byte) []byte {
	if len(htmlpage) == 0 {
		return EchartsJS()
	}
	s := bytes.ReplaceAll(htmlpage, json.Bytes(`<script src="https://go-echarts.github.io/go-echarts-assets/assets/echarts.min.js"></script>`), EchartsJS())
	s = bytes.ReplaceAll(s, json.Bytes(`<script src="https://go-echarts.github.io/go-echarts-assets/assets/themes/themeRiver.js"></script>`), []byte{})
	return s
}
