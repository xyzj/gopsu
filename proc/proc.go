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
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/cache"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
)

// go-echats should be v2.3.3

//go:embed echarts.min.js
var echarts []byte

type procStatus struct {
	Memrss  uint64  `json:"rss"`
	Memvms  uint64  `json:"vms"`
	IORead  uint64  `json:"ior"`
	IOWrite uint64  `json:"iow"`
	Dt      int64   `json:"dt"`
	Cpup    float32 `json:"cpu"`
	Memp    float32 `json:"mem"`
	Ofd     int32   `json:"ofd"`
	Conns   int32   `json:"conn"`
}

func (ps *procStatus) String() string {
	return fmt.Sprintf("cpu: %.2f%%; mem: %.2f%%; rss: %s", ps.Cpup, ps.Memp, gopsu.FormatFileSize(ps.Memrss))
}

func (ps *procStatus) HTML() string {
	return fmt.Sprintf("cpu: %.2f%%\nmem: %.2f%%\nrss: %s\nofd: %d", ps.Cpup, ps.Memp, gopsu.FormatFileSize(ps.Memrss), ps.Ofd)
}

func (ps *procStatus) JSON() string {
	js, err := json.MarshalToString(ps)
	if err != nil {
		return ""
	}
	// js, _ = sjson.Set(js, "dt", gopsu.Stamp2Time(ps.Dt))
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
		var proce *process.Process
		var err error
		var memi *process.MemoryInfoStat
		var iost *process.IOCountersStat
		var connst []net.ConnectionStat
		var cp float64
		f := func() {
			if proce == nil {
				proce, err = process.NewProcess(int32(os.Getpid()))
				if err != nil {
					opt.Logg.Error("[PROC] " + err.Error())
					return
				}
			}
			cp, _ = proce.CPUPercent()
			r.lastProc.Cpup = float32(cp)
			r.lastProc.Ofd, _ = proce.NumFDs()
			r.lastProc.Memp, _ = proce.MemoryPercent()
			memi, _ = proce.MemoryInfo()
			if memi == nil {
				memi = &process.MemoryInfoStat{}
			}
			r.lastProc.Memrss = memi.RSS
			r.lastProc.Memvms = memi.VMS
			iost, _ = proce.IOCounters()
			if iost == nil {
				iost = &process.IOCountersStat{}
			}
			r.lastProc.IORead = iost.ReadBytes
			r.lastProc.IOWrite = iost.WriteBytes
			connst, _ = proce.Connections()
			r.lastProc.Conns = int32(len(connst))
			r.lastProc.Dt = time.Now().Unix()
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

func (r *Recorder) Import(s []byte) {
	gjson.ParseBytes(s).Get("data").ForEach(func(key, value gjson.Result) bool {
		ls := &procStatus{
			Dt:      value.Get("dt").Int(),
			Cpup:    float32(value.Get("cpu").Float()),
			Memp:    float32(value.Get("mem").Float()),
			Memrss:  value.Get("rss").Uint(),
			Memvms:  value.Get("vms").Uint(),
			Ofd:     int32(value.Get("ofd").Int()),
			Conns:   int32(value.Get("conn").Int()),
			IORead:  value.Get("ior").Uint(),
			IOWrite: value.Get("iow").Uint(),
		}
		r.procCache.Store(gopsu.Stamp2Time(ls.Dt, "01-02 15:04:05"), ls)
		return true
	})
}

func (r *Recorder) Export() []byte {
	s := []byte{}
	for _, v := range r.allData() {
		s, _ = sjson.SetBytes(s, "data.-1", v)
	}
	return s
}

// allData 返回所有数据
func (r *Recorder) allData() []*procStatus {
	js := make([]*procStatus, 0, r.procCache.Len())
	r.procCache.ForEach(func(key string, value *procStatus) bool {
		js = append(js, value)
		return true
	})
	sort.Slice(js, func(i, j int) bool {
		return js[i].Dt < js[j].Dt
	})
	return js
}

func (r *Recorder) BuildLines(width string, js []*procStatus) []byte {
	var nametail string
	if r.opt.Name != "" {
		nametail = " (" + r.opt.Name + ")"
	}
	l := len(js)
	x := make([]string, 0, l)
	cpu := make([]opts.LineData, 0, l)
	mem := make([]opts.LineData, 0, l)
	rss := make([]opts.LineData, 0, l)
	vms := make([]opts.LineData, 0, l)
	ofd := make([]opts.LineData, 0, l)
	ior := make([]opts.LineData, 0, l)
	iow := make([]opts.LineData, 0, l)
	con := make([]opts.LineData, 0, l)
	for _, v := range js {
		x = append(x, gopsu.Stamp2Time(v.Dt, "01-02 15:04:05"))
		cpu = append(cpu, opts.LineData{Name: fmt.Sprintf("%.2f%%", v.Cpup), Value: v.Cpup, Symbol: "circle"})
		mem = append(mem, opts.LineData{Name: fmt.Sprintf("%.2f%%", v.Memp), Value: v.Memp, Symbol: "circle"})
		rss = append(rss, opts.LineData{Name: gopsu.FormatFileSize(v.Memrss), Value: v.Memrss / 1024 / 1024, Symbol: "circle"})
		vms = append(vms, opts.LineData{Name: gopsu.FormatFileSize(v.Memvms), Value: v.Memvms / 1024 / 1024, Symbol: "circle"})
		ofd = append(ofd, opts.LineData{Name: fmt.Sprintf("%d", v.Ofd), Value: v.Ofd, Symbol: "circle"})
		ior = append(ior, opts.LineData{Name: gopsu.FormatFileSize(v.IORead), Value: v.IORead / 1024 / 1024, Symbol: "circle"})
		iow = append(iow, opts.LineData{Name: gopsu.FormatFileSize(v.IOWrite), Value: v.IOWrite / 1024 / 1024, Symbol: "circle"})
		con = append(con, opts.LineData{Name: fmt.Sprintf("%d", v.Conns), Value: v.Conns, Symbol: "circle"})
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
	line1.SetSeriesOptions(SetupLineSOpts()...)
	line1.AddSeries("cpu", cpu)
	line1.AddSeries("mem", mem)

	lineMem := charts.NewLine()
	lineMem.SetGlobalOptions(SetupLineGOpts(&LineOpt{
		Name:        "Resident Set Size & Virtual Memory Size" + nametail,
		Total:       float32(l),
		TTFormatter: "{a0}: <b>{b0}</b><br>{a1}: <b>{b1}</b>",
		YFormatter:  "{value} MB",
		Width:       width,
	})...)
	lineMem.YAxisList = []opts.YAxis{
		{
			Name:        "rss",
			Show:        true,
			SplitNumber: 7,
			SplitLine: &opts.SplitLine{
				Show: false,
			},
			SplitArea: &opts.SplitArea{
				Show: true,
			},
			AxisLabel: &opts.AxisLabel{
				Show:      true,
				Formatter: "{value} MB",
				Align:     "left",
			},
		},
		{
			Name:        "vms",
			Show:        true,
			SplitNumber: 7,
			SplitLine: &opts.SplitLine{
				Show: false,
			},
			SplitArea: &opts.SplitArea{
				Show: false,
			},
			AxisLabel: &opts.AxisLabel{
				Show:      true,
				Formatter: "{value} MB",
				Align:     "right",
			},
		},
	}
	lineMem.SetXAxis(x)
	lineMem.AddSeries("rss", rss, charts.WithLineChartOpts(
		opts.LineChart{
			Smooth:     true,
			YAxisIndex: 0,
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Width: 2,
		}), charts.WithAreaStyleOpts(opts.AreaStyle{
			Opacity: 0.3,
		}))
	lineMem.AddSeries("vms", vms, charts.WithLineChartOpts(
		opts.LineChart{
			Smooth:     false,
			YAxisIndex: 1,
		}))

	lineOfd := charts.NewLine()
	lineOfd.SetGlobalOptions(SetupLineGOpts(&LineOpt{
		Name:        "Opened File Descriptors & Connections" + nametail,
		Total:       float32(l),
		TTFormatter: "{a0}: <b>{b0}</b><br>{a1}: <b>{b1}</b>",
		Width:       width,
	})...)
	lineOfd.SetXAxis(x)
	lineOfd.SetSeriesOptions(SetupLineSOpts(charts.WithAreaStyleOpts(opts.AreaStyle{
		Opacity: 0.2,
	}))...)
	lineOfd.AddSeries("ofd", ofd)
	lineOfd.AddSeries("conn", con)

	lineIO := charts.NewLine()
	lineIO.SetGlobalOptions(SetupLineGOpts(&LineOpt{
		Name:        "IO Counts" + nametail,
		Total:       float32(l),
		TTFormatter: "{a0}: <b>{b0}</b><br>{a1}: <b>{b1}</b>",
		YFormatter:  "{value} MB",
		Width:       width,
	})...)
	lineIO.SetXAxis(x)
	lineIO.SetSeriesOptions(SetupLineSOpts(charts.WithAreaStyleOpts(opts.AreaStyle{
		Opacity: 0.3,
	}))...)
	lineIO.AddSeries("read", ior)
	lineIO.AddSeries("write", iow)
	a := components.NewPage()
	a.PageTitle = "Process Records" + nametail
	a.AddCharts(line1, lineOfd, lineMem, lineIO)
	b := &bytes.Buffer{}
	a.Render(b)
	return b.Bytes()
}

func (r *Recorder) GinHandler(c *gin.Context) {
	js := r.allData()
	switch c.Request.Method {
	case "POST":
		c.Set("data", js)
		c.Set("status", 1)
		c.JSON(200, c.Keys)
	case "GET":
		c.Writer.Write(LocalEchartsJS(r.BuildLines(c.Param("width"), js)))
	}
}

func (r *Recorder) HTTPHandler(w http.ResponseWriter, req *http.Request) {
	js := r.allData()
	switch req.Method {
	case "POST":
		s, _ := sjson.SetBytes([]byte{}, "status", 1)
		s, _ = sjson.SetBytes(s, "data", js)
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(s)
	case "GET":
		values := req.URL.Query()
		w.Write(LocalEchartsJS(r.BuildLines(values.Get("width"), js)))
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
		charts.WithToolboxOpts(opts.Toolbox{
			Show: true,
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
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
			SplitLine: &opts.SplitLine{
				Show: false,
			},
			SplitArea: &opts.SplitArea{
				Show: true,
			},
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
