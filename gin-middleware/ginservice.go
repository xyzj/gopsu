// Package ginmiddleware 基于gin的web框架封装
package ginmiddleware

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/loopfunc"
	"github.com/xyzj/gopsu/pathtool"
)

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
	Engine       *gin.Engine
	Hosts        []string
	CertFile     string
	KeyFile      string
	LogFile      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	HTTPPort     string
	HTTPSPort    string
	LogDays      int
	Debug        bool
}

// ListenAndServe 启用监听
// port：端口号
// h： http.hander, like gin.New()
func ListenAndServe(port int, h *gin.Engine) error {
	ListenAndServeWithOption(&ServiceOption{
		HTTPPort:   fmt.Sprintf(":%d", port),
		EngineFunc: func() *gin.Engine { return h },
	})
	return nil
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
// clientca: 客户端根证书用于验证客户端合法性
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, clientca ...string) error {
	ListenAndServeWithOption(&ServiceOption{
		EngineFunc: func() *gin.Engine { return h },
		HTTPSPort:  fmt.Sprintf(":%d", port),
		CertFile:   certfile,
		KeyFile:    keyfile,
	})
	return nil
}

// ListenAndServeWithOption 启动服务
func ListenAndServeWithOption(opt *ServiceOption) {
	if opt.HTTPPort == "" && opt.HTTPSPort == "" {
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
			return LiteEngine(opt.LogFile, opt.LogDays, opt.Hosts...)
		}
	}
	// 路由处理
	var findRoot = false
	var findIcon = false
	h := opt.EngineFunc()
	for _, v := range h.Routes() {
		if v.Path == "/" {
			findRoot = true
			continue
		}
		if v.Path == "/favicon.ico" {
			findIcon = true
		}
		if findRoot && findIcon {
			break
		}
	}
	if !findRoot {
		h.GET("/", PageDefault)
	}
	if !findIcon {
		h.GET("/favicon.ico", func(c *gin.Context) {
			c.Writer.Write(favicon)
		})
	}
	opt.Engine = h

	// 启动https服务
	if opt.HTTPSPort != ":0" && opt.HTTPSPort != "" {
		loopfunc.GoFunc(func(params ...interface{}) {
			if !pathtool.IsExist(opt.CertFile) || !pathtool.IsExist(opt.KeyFile) {
				fmt.Fprintf(os.Stdout, "%s [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "HTTPS server error: no cert or key file found")
				return
			}
			cc, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile)
			if err != nil {
				fmt.Fprintf(os.Stdout, "%s [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "cert and key file load error:"+err.Error())
				return
			}
			s := &http.Server{
				Addr:         opt.HTTPSPort,
				ReadTimeout:  opt.ReadTimeout,
				WriteTimeout: opt.WriteTimeout,
				IdleTimeout:  opt.IdleTimeout,
				Handler:      h,
				TLSConfig: &tls.Config{
					Certificates: []tls.Certificate{cc},
					CipherSuites: []uint16{
						tls.TLS_AES_128_GCM_SHA256,
						tls.TLS_AES_256_GCM_SHA384,
						tls.TLS_CHACHA20_POLY1305_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
					},
				},
			}
			loopfunc.GoFunc(func(params ...interface{}) {
				for {
					time.Sleep(time.Hour * 23)
					if cc, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile); err == nil {
						s.TLSConfig.Certificates[0] = cc
						// s.TLSConfig = &tls.Config{
						// 	Certificates: []tls.Certificate{cc},
						// }
					}
				}
			}, "cert update", os.Stdout)
			fmt.Fprintf(os.Stdout, "%s [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTPS server at "+opt.HTTPSPort)
			if err := s.ListenAndServeTLS("", ""); err != nil {
				fmt.Fprintf(os.Stdout, "%s [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTPS server error: "+err.Error())
			}
		}, "https", os.Stdout)
	}
	// 启动http服务
	if opt.HTTPPort != ":0" && opt.HTTPPort != "" {
		loopfunc.GoFunc(func(params ...interface{}) {
			s := &http.Server{
				Addr:         opt.HTTPPort,
				ReadTimeout:  opt.ReadTimeout,
				WriteTimeout: opt.WriteTimeout,
				IdleTimeout:  opt.IdleTimeout,
				Handler:      h,
			}
			fmt.Fprintf(os.Stdout, "%s [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTP server at "+opt.HTTPPort)
			if err := s.ListenAndServe(); err != nil {
				fmt.Fprintf(os.Stdout, "%s [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Start HTTP server error: "+err.Error())
			}
		}, "http", os.Stdout)
	}
	select {}
}

// LiteEngine 轻量化基础引擎
func LiteEngine(logfile string, logDays int, hosts ...string) *gin.Engine {
	r := gin.New()
	// 特殊路由处理
	r.HandleMethodNotAllowed = true
	r.NoMethod(Page405)
	r.NoRoute(Page404Big)
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
	// 处理转发ip
	r.Use(XForwardedIP())
	// 配置日志
	var logDir, logName string
	if logfile != "" && logDays > 0 {
		logDir, logName = filepath.Split(logfile)
	}
	r.Use(LoggerWithRolling(logDir, logName, logDays))
	// 故障恢复
	r.Use(Recovery())
	// 绑定域名
	r.Use(bindHosts(hosts...))
	// 数据压缩
	// r.Use(gingzip.Gzip(6))
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
