package ginmiddleware

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/xyzj/gopsu"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

/*
// ServiceProtocol http协议类型
type ServiceProtocol int

const (
	// ProtocolHTTP http协议
	ProtocolHTTP ServiceProtocol = iota
	// ProtocolHTTPS https协议
	ProtocolHTTPS
	// PtorocolBoth 2种协议
	PtorocolBoth
)
*/

// ServiceOption 通用化http框架
type ServiceOption struct {
	EngineFunc   func() *gin.Engine
	CertFile     string
	KeyFile      string
	HTTPPort     int
	HTTPSPort    int
	Host         []string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Debug        bool
}

// ListenAndServeWithOption 启动服务
func ListenAndServeWithOption(opt *ServiceOption) {
	if opt.HTTPPort == 0 && opt.HTTPSPort == 0 {
		println("no server start")
		os.Exit(1)
	}
	if !opt.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	if opt.ReadTimeout == 0 {
		opt.ReadTimeout = time.Second * 120
	}
	if opt.WriteTimeout == 0 {
		opt.WriteTimeout = time.Second * 120
	}
	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = time.Second * 60
	}
	if opt.EngineFunc == nil {
		opt.EngineFunc = func() *gin.Engine {
			r := gin.New()
			// 允许跨域
			r.Use(cors.New(cors.Config{
				MaxAge:           time.Hour * 24,
				AllowAllOrigins:  true,
				AllowCredentials: true,
				AllowWildcard:    true,
				AllowMethods:     []string{"*"},
				AllowHeaders:     []string{"*"},
			}))
			LoggerWithRolling(gopsu.GetExecDir(), "", 0)
			// 故障恢复
			r.Use(Recovery())
			// 特殊路由处理
			r.HandleMethodNotAllowed = true
			r.NoMethod(Page405)
			r.NoRoute(Page404)
			return r
		}
	}
	// 路由处理
	var findRoot = false
	h := opt.EngineFunc()
	for _, v := range h.Routes() {
		if v.Path == "/" {
			findRoot = true
			break
		}
	}
	if !findRoot {
		h.GET("/", PageDefault)
	}
	// 启动https服务
	if opt.HTTPSPort > 0 {
		go func() {
			var tc *tls.Config
			s := &http.Server{
				Addr:         fmt.Sprintf(":%d", opt.HTTPSPort),
				ReadTimeout:  opt.ReadTimeout,
				WriteTimeout: opt.WriteTimeout,
				IdleTimeout:  opt.IdleTimeout,
				Handler:      h,
				TLSConfig:    tc,
			}
			if len(opt.CertFile) > 0 &&
				len(opt.KeyFile) > 0 &&
				gopsu.IsExist(opt.CertFile) &&
				gopsu.IsExist(opt.KeyFile) {
				go func() {
					defer func() {
						if err := recover(); err != nil {
							fmt.Fprintf(os.Stdout, "cert update crash: %s\n", err.(error).Error())
						}
					}()

					if cc, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile); err == nil {
						s.TLSConfig = &tls.Config{
							Certificates: []tls.Certificate{cc},
						}
					}
					time.Sleep(time.Hour * time.Duration(1+rand.Int31n(5)))
				}()
				time.Sleep(time.Second)
			} else {
				fmt.Fprintf(os.Stdout, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "HTTPS server error: no cert or key file found")
				return
			}
			fmt.Fprintf(os.Stdout, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTPS server at :"+strconv.Itoa(opt.HTTPSPort))
			if err := s.ListenAndServeTLS("", ""); err != nil {
				fmt.Fprintf(os.Stdout, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTPS server error: "+err.Error())
			}
		}()
	}
	// 启动http服务
	if opt.HTTPPort > 0 {
		go func() {
			s := &http.Server{
				Addr:         fmt.Sprintf(":%d", opt.HTTPPort),
				ReadTimeout:  opt.ReadTimeout,
				WriteTimeout: opt.WriteTimeout,
				IdleTimeout:  opt.IdleTimeout,
				Handler:      h,
			}
			fmt.Fprintf(os.Stdout, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTP server at :"+strconv.Itoa(opt.HTTPPort))
			if err := s.ListenAndServe(); err != nil {
				fmt.Fprintf(os.Stdout, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTP server error: "+err.Error())
			}
		}()
	}
	for {
		time.Sleep(time.Hour)
	}
}

// LiteEngine 轻量化基础引擎
func LiteEngine(logDir, logName string, logDays int, hosts ...string) *gin.Engine {
	r := gin.New()
	// 允许跨域
	r.Use(cors.New(cors.Config{
		MaxAge:           time.Hour * 24,
		AllowWebSockets:  true,
		AllowCredentials: true,
		AllowWildcard:    true,
		AllowAllOrigins:  true,
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
	}))
	// 配置日志
	LoggerWithRolling(logDir, logName, logDays)
	// 故障恢复
	r.Use(Recovery())
	// 绑定域名
	r.Use(bindHosts(hosts...))
	// 特殊路由处理
	r.HandleMethodNotAllowed = true
	r.NoMethod(Page405)
	r.NoRoute(Page404)
	return r
}

func bindHosts(hosts ...string) gin.HandlerFunc {
	if len(hosts) == 0 {
		return func(c *gin.Context) {}
	}
	return func(c *gin.Context) {
		host, _, _ := net.SplitHostPort(c.Request.Host)
		nohost := true
		for _, v := range hosts {
			if v == host {
				nohost = false
				break
			}
		}
		if nohost {
			c.Set("status", 0)
			c.Set("detail", "forbidden")
			c.AbortWithStatusJSON(http.StatusForbidden, c.Keys)
		}
	}
}
