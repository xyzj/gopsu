package ginmiddleware

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/xyzj/gopsu"
)

var (
	template404 = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>404</title><style>    @import url(https://fonts.googleapis.com/css?family=Exo+2:200i);:root{font-size:10px;--neon-text-color:#f40;--neon-border-color:#08f}body{display:flex;margin:0;padding:0;min-height:100vh;border:0;background:#000;font-size:100%;font-family:'Exo 2',sans-serif;line-height:1;justify-content:center;align-items:center}h1{padding:4rem 6rem 5.5rem;border:.4rem solid #fff;border-radius:2rem;color:#fff;text-transform:uppercase;font-weight:200;font-style:italic;font-size:13rem;animation:flicker 1s infinite alternate}h1::-moz-selection{background-color:var(--neon-border-color);color:var(--neon-text-color)}h1::selection{background-color:var(--neon-border-color);color:var(--neon-text-color)}h1:focus{outline:0}.flicker-text-fast{animation:flicker-text 1.5s infinite alternate}.flicker-text-slow{animation:flicker-text 4.4s infinite alternate}@keyframes flicker{0%{box-shadow:0 0 .5rem #fff,inset 0 0 .5rem #fff,0 0 2rem var(--neon-border-color),inset 0 0 2rem var(--neon-border-color),0 0 4rem var(--neon-border-color),inset 0 0 4rem var(--neon-border-color);text-shadow:-.2rem -.2rem 1rem #fff,.2rem .2rem 1rem #fff,0 0 2rem var(--neon-text-color),0 0 4rem var(--neon-text-color),0 0 6rem var(--neon-text-color),0 0 8rem var(--neon-text-color),0 0 10rem var(--neon-text-color)}100%{box-shadow:0 0 .5rem #fff,inset 0 0 .5rem #fff,0 0 2rem var(--neon-border-color),inset 0 0 2rem var(--neon-border-color),0 0 4rem var(--neon-border-color),inset 0 0 4rem var(--neon-border-color);text-shadow:-.2rem -.2rem 1rem #fff,.2rem .2rem 1rem #fff,0 0 2rem var(--neon-text-color),0 0 4rem var(--neon-text-color),0 0 6rem var(--neon-text-color),0 0 8rem var(--neon-text-color),0 0 10rem var(--neon-text-color)}}@keyframes flicker-text{0%,100%,19%,21%,23%,25%,54%,56%{box-shadow:none;text-shadow:-.2rem -.2rem 1rem #fff,.2rem .2rem 1rem #fff,0 0 2rem var(--neon-text-color),0 0 4rem var(--neon-text-color),0 0 6rem var(--neon-text-color),0 0 8rem var(--neon-text-color),0 0 10rem var(--neon-text-color)}20%,24%,55%{box-shadow:none;text-shadow:none}}</style></head><body><h1 contenteditable spellcheck="false"><span class="flicker-text-slow">40</span><span class="flicker-text-fast">4</span></h1></body></html>`

	templateHelloWorld = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Aloha</title><style>@import url(https://fonts.googleapis.com/css?family=Montserrat:700);body{margin:0;width:100%;height:100vh;overflow:hidden;background:hsla(0,5%,5%,1);background-repeat:no-repeat;background-attachment:fixed;background-image:-webkit-gradient(linear,left bottom,right top,from(hsla(0,5%,15%,0.5)),to(hsla(0,5%,5%,1)));background-image:linear-gradient(to right top,hsla(0,5%,15%,0.5),hsla(0,5%,5%,1))}svg{width:100%}</style></head><body><svg width="100%" height="100%" viewBox="30 -50 600 500" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1"><path id="path"><animate attributeName="d" from="m0,110 h0" to="m0,110 h1100" dur="7s" begin="0.5s" repeatCount="indefinite"/></path><text font-size="30" font-family="Montserrat" fill='hsla(36, 95%, 85%, 1)'><textPath xlink:href="#path">Aloha Golang 🍹</textPath></text></svg></body></html>`

	templateRuntime = `<html lang="zh-cn">
<head>
    <meta content="text/html; charset=utf-8" http-equiv="content-type" />
    <!-- <script language="JavaScript">
	      function myrefresh()
	        {
	          window.location.reload();
	        }
	      setTimeout('myrefresh()',180000); //指定180s刷新一次
	    </script> -->
    <style type="text/css">
        a {
            color: #4183C4;
            font-size: 16px;
        }

        h1,
        h2,
        h3,
        h4,
        h5,
        h6 {
            margin: 20px 0 10px;
            padding: 0;
            font-weight: bold;
            -webkit-font-smoothing: antialiased;
            cursor: text;
            position: relative;
        }

        h1 {
            font-size: 28px;
            color: black;
        }

        h2 {
            font-size: 24px;
            border-bottom: 1px solid #cccccc;
            color: black;
        }

        h3 {
            font-size: 18px;
        }

        h4 {
            font-size: 16px;
        }

        h5 {
            font-size: 14px;
        }

        h6 {
            color: #777777;
            font-size: 14px;
        }

        table {
            padding: 0;
        }

        table tr {
            border-top: 1px solid #cccccc;
            background-color: white;
            margin: 0;
            padding: 0;
        }

        table tr:nth-child(2n) {
            background-color: #f8f8f8;
        }

        table tr th {
            font-weight: bold;
            border: 1px solid #cccccc;
            text-align: center;
            margin: 0;
            padding: 6px 13px;
        }

        table tr td {
            border: 1px solid #cccccc;
            text-align: center;
            margin: 0;
            padding: 6px 13px;
        }

        table tr th :first-child,
        table tr td :first-child {
            margin-top: 0;
        }

        table tr th :last-child,
        table tr td :last-child {
            margin-bottom: 0;
        }
    </style>
</head>

<body>
    <h3>服务器时间：</h3><a>{{.timer}}</a>
    <h3>服务软件信息：</h3><a>{{range $idx, $elem := .ver}}
        {{$elem}} <br>
        {{end}}</a>
</body>

</html>`
)

var (
	runtimeInfo map[string]interface{}
)

func init() {
	runtimeInfo = make(map[string]interface{})
	runtimeInfo["timer"] = time.Now().Format("2006-01-02 15:04:05 Mon")
	runtimeInfo["ver"] = []string{}
}

// Page404 Page404
func Page404(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Status(http.StatusNotFound)
	render.WriteString(c.Writer, template404, nil)
}

// Page405 Page405
func Page405(c *gin.Context) {
	c.String(http.StatusMethodNotAllowed, "method not allowed")
}

// PageDefault 健康检查
func PageDefault(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		if c.Request.RequestURI == "/" {
			c.Header("Content-Type", "text/html")
			c.Status(http.StatusOK)
			render.WriteString(c.Writer, templateHelloWorld, nil)
		} else {
			c.String(200, "ok")
		}
	case "POST":
		c.String(200, "ok")
	}
}

// PageRuntime 启动信息显示
func PageRuntime(c *gin.Context) {
	if len(runtimeInfo["ver"].([]string)) == 0 {
		_, fn, _, ok := runtime.Caller(0)
		if ok {
			b, err := ioutil.ReadFile(path.Base(fn) + ".ver")
			if err == nil {
				runtimeInfo["ver"] = strings.Split(string(b), "\n")
			}
		}
	}
	switch c.Request.Method {
	case "GET":
		// c.Header("Content-Type", "text/html")
		// c.Status(http.StatusOK)
		// render.WriteString(c.Writer, templateRuntime, nil)
		t, _ := template.New("runtime").Parse(templateRuntime)
		h := render.HTML{
			Name:     "runtime",
			Data:     runtimeInfo,
			Template: t,
		}
		h.WriteContentType(c.Writer)
		h.Render(c.Writer)
	case "POST":
		c.PureJSON(200, runtimeInfo)
	}
}

// Clearlog 日志清理
func Clearlog(c *gin.Context) {
	if c.Param("pwd") != "xyissogood" {
		c.String(200, "Wrong!!!")
		return
	}
	var days int64
	if days = gopsu.String2Int64(c.Param("days"), 0); days == 0 {
		days = 7
	}
	// 遍历文件夹
	dir := c.Param("dir")
	if dir == "" {
		dir = gopsu.DefaultLogDir
	}
	lstfno, ex := ioutil.ReadDir(dir)
	if ex != nil {
		ioutil.WriteFile("ginlogerr.log", []byte(fmt.Sprintf("clear log files error: %s", ex.Error())), 0664)
	}
	t := time.Now()
	for _, fno := range lstfno {
		if fno.IsDir() || !strings.Contains(fno.Name(), c.Param("name")) { // 忽略目录，不含日志名的文件，以及当前文件
			continue
		}
		// 比对文件生存期
		if t.Unix()-fno.ModTime().Unix() >= days*24*60*60-10 {
			os.Remove(filepath.Join(c.Param("dir"), fno.Name()))
			c.Set(fno.Name(), "deleted")
		}
	}
	c.PureJSON(200, c.Keys)
}

// SetVersionInfo 设置服务版本信息
func SetVersionInfo(ver string) {
	runtimeInfo["ver"] = strings.Split(ver, "\n")[1:]
}
