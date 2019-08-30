package microgo

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/pkg/transport"
)

const (
	leaseTimeout   = 7
	contextTimeout = 2 * time.Second
)

// Etcdv3Client 微服务结构体
type Etcdv3Client struct {
	etcdLog      *io.Writer       // 日志
	etcdLogLevel int              // 日志等级
	etcdRoot     string           // etcd注册根路经
	etcdAddr     []string         // etcd服务地址
	etcdClient   *clientv3.Client // 连接实例
	svrName      string           // 服务名称
	svrPool      sync.Map         // 线程安全服务信息字典
	svrDetail    string           // 服务信息
}

// RegisteredServer 获取到的服务注册信息
type registeredServer struct {
	svrName       string // 服务名称
	svrAddr       string // 服务地址
	svrPickTimes  int    // 命中次数
	svrProtocol   string // 服务使用数据格式
	svrInterface  string // 服务发布的接口类型
	svrActiveTime int64  // 服务查询时间
}

// NewEtcdv3Client 获取新的微服务结构
func NewEtcdv3Client(etcdaddr []string) (*Etcdv3Client, error) {
	return NewEtcdv3ClientTLS(etcdaddr, "", "", "")
}

// NewEtcdv3ClientTLS 获取新的微服务结构（tls）
func NewEtcdv3ClientTLS(etcdaddr []string, certfile, keyfile, cafile string) (*Etcdv3Client, error) {
	m := &Etcdv3Client{
		etcdRoot: "wlst-micro",
		etcdAddr: etcdaddr,
	}
	var tlsconf *tls.Config
	if gopsu.IsExist(certfile) && gopsu.IsExist(keyfile) && gopsu.IsExist(cafile) {
		tlsinfo := transport.TLSInfo{
			CertFile:      certfile,
			KeyFile:       keyfile,
			TrustedCAFile: cafile,
		}
		var err error
		tlsconf, err = tlsinfo.ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		tlsconf = nil
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   m.etcdAddr,
		DialTimeout: 2 * time.Second,
		TLS:         tlsconf,
	})
	if err != nil {
		return nil, err
	}
	m.etcdClient = cli
	return m, nil
}

func (m *Etcdv3Client) writeLog(s string, level int) {
	s = fmt.Sprintf("%v [%02d] [ETCD] %s", time.Now().Format(gopsu.LogTimeFormat), level, s)
	if m.etcdLog == nil {
		println(s)
	} else {
		if level >= m.etcdLogLevel && level < 90 {
			fmt.Fprintln(*m.etcdLog, s)
		} else if level == 90 {
			println(s)
		}
	}
}

// listServers 查询根路径下所有服务
func (m *Etcdv3Client) listServers() error {
	defer func() error {
		if err := recover(); err != nil {
			// fmt.Printf("%+v\n", err)
		}
		return nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	resp, err := m.etcdClient.Get(ctx, fmt.Sprintf("/%s", m.etcdRoot), clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}
	for _, v := range resp.Kvs {
		va := gjson.ParseBytes(v.Value)
		s := &registeredServer{
			svrName:       va.Get("name").String(),
			svrAddr:       fmt.Sprintf("%s:%s", va.Get("ip").String(), va.Get("port").String()),
			svrPickTimes:  0,
			svrProtocol:   va.Get("protocol").String(),
			svrInterface:  va.Get("INTFC").String(),
			svrActiveTime: time.Now().Unix(),
		}
		if s.svrName == "" {
			x := strings.Split(string(v.Key), "/")
			if len(x) > 2 {
				s.svrName = x[1]
			}
		}
		a, ok := m.svrPool.LoadOrStore(string(v.Key), s)
		if ok {
			s := a.(*registeredServer)
			s.svrActiveTime = time.Now().Unix()
			m.svrPool.Store(string(v.Key), s)
		}
	}
	return nil
}

// addPickTimes 增加计数器
func (m *Etcdv3Client) addPickTimes(k string, r *registeredServer) {
	if r.svrPickTimes >= 0xffffff { // 防止溢出
		r.svrPickTimes = 0
	} else {
		r.svrPickTimes++
	}
	m.svrPool.Store(k, r)
}

// 服务注册
func (m *Etcdv3Client) etcdRegister() (*clientv3.LeaseID, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	lresp, err := m.etcdClient.Grant(ctx, leaseTimeout)
	defer cancel()
	if err != nil {
		m.writeLog(fmt.Sprintf("Registration to %s failed: %v", m.etcdAddr, err.Error()), 40)
		return nil, false
	}
	m.etcdClient.Put(ctx, fmt.Sprintf("/%s/%s/%s_%s", m.etcdRoot, m.svrName, m.svrName, gopsu.GetUUID1()), m.svrDetail, clientv3.WithLease(lresp.ID))
	m.writeLog(fmt.Sprintf("Registration to %v success.", m.etcdAddr), 90)
	return &lresp.ID, true
}

// SetRoot 自定义根路径
//
// args:
//  root: 注册根路径，默认'wlst-micro'
func (m *Etcdv3Client) SetRoot(root string) {
	m.etcdRoot = root
}

// SetLogger 设置日志记录器
func (m *Etcdv3Client) SetLogger(l *io.Writer, level int) {
	m.etcdLog = l
	m.etcdLogLevel = level
}

// Register 服务注册
//
// args:
//  svrname: 服务名称
//  svrip: 服务ip
//  intfc: 接口类型
//  protoname: 协议类型
//  svrport: 服务端口
// return:
//  error
func (m *Etcdv3Client) Register(svrname, svrip, svrport, intfc, protoname string) {
	m.svrName = svrname
	if svrip == "" {
		svrip, _ = gopsu.RealIP("")
	}
	js, _ := sjson.Set("", "ip", svrip)
	js, _ = sjson.Set(js, "port", svrport)
	js, _ = sjson.Set(js, "name", svrname)
	js, _ = sjson.Set(js, "INTFC", intfc)
	js, _ = sjson.Set(js, "protocol", protoname)
	js, _ = sjson.Set(js, "timeConnect", time.Now().Unix())
	js, _ = sjson.Set(js, "timeActive", time.Now().Unix())
	m.svrDetail = js

	// 监视线程，在etcd崩溃并重启时重新注册
	go func() {
		// 注册
		leaseid, ok := m.etcdRegister()
		// 使用1-4s内的随机间隔
		t := time.NewTicker(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
		for _ = range t.C {
			if ok { // 成功注册时发送心跳
				ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
				_, err := m.etcdClient.KeepAliveOnce(ctx, *leaseid)
				cancel()
				if err != nil {
					m.writeLog("Lost connection with etcd server, retrying ...", 40)
					ok = false
				}
			} else { // 注册失败时重新注册
				leaseid, ok = m.etcdRegister()
			}
		}
	}()
}

// Watcher 监视服务信息变化
func (m *Etcdv3Client) Watcher(model ...byte) error {
	m.listServers()
	mo := byte(0)
	if len(model) > 0 {
		mo = model[0]
	}
	switch mo {
	default: // 默认采用定时主动获取
		go func() {
			for {
				select {
				case <-time.Tick(time.Second * 2):
					m.listServers()
				}
			}
		}()
	}
	return nil
}

func (m *Etcdv3Client) pickerList(svrname string, intfc ...string) [][]string {
	t := time.Now().Unix()
	listSvr := make([][]string, 0)
	// 找到所有同名服务
	switch len(intfc) {
	case 1: // 匹配服务名称和接口类型
		m.svrPool.Range(func(k, v interface{}) bool {
			s := v.(*registeredServer)
			// 删除无效服务信息
			if t-s.svrActiveTime >= 5 {
				m.svrPool.Delete(k)
				return true
			}
			if s.svrName == svrname && s.svrInterface == intfc[0] {
				listSvr = append(listSvr, []string{fmt.Sprintf("%012d", s.svrPickTimes), s.svrAddr, k.(string)})
			}
			return true
		})
	case 2: // 匹配服务名称，接口类型，和协议类型
		m.svrPool.Range(func(k, v interface{}) bool {
			s := v.(*registeredServer)
			// 删除无效服务信息
			if t-s.svrActiveTime >= 5 {
				m.svrPool.Delete(k)
				return true
			}
			if s.svrName == svrname && s.svrInterface == intfc[0] && s.svrProtocol == intfc[1] {
				listSvr = append(listSvr, []string{fmt.Sprintf("%012d", s.svrPickTimes), s.svrAddr, k.(string)})
			}
			return true
		})
	default: // 仅匹配服务名称
		m.svrPool.Range(func(k, v interface{}) bool {
			s := v.(*registeredServer)
			// 删除无效服务信息
			if t-s.svrActiveTime >= 5 {
				m.svrPool.Delete(k)
				return true
			}
			if s.svrName == svrname {
				listSvr = append(listSvr, []string{fmt.Sprintf("%012d", s.svrPickTimes), s.svrAddr, k.(string)})
			}
			return true
		})
	}
	return listSvr
}

// PickerAll 服务选择
//
// args:
//  svrname: 服务名称
//  intfc: 服务类型，协议类型
// return:
//  string: 服务地址
//  error
func (m *Etcdv3Client) PickerAll(svrname string, intfc ...string) []string {
	listSvr := m.pickerList(svrname, intfc...)
	var allSvr = make([]string, 0)
	for _, v := range listSvr {
		allSvr = append(allSvr, v[2])
	}
	return allSvr
}

// Picker 服务选择
//
// args:
//  svrname: 服务名称
//  intfc: 服务类型，协议类型
// return:
//  string: 服务地址
//  error
func (m *Etcdv3Client) Picker(svrname string, intfc ...string) (string, error) {
	listSvr := m.pickerList(svrname, intfc...)
	if len(listSvr) > 0 {
		// 排序，获取命中最少的服务地址
		sortlist := &gopsu.StringSliceSort{}
		sortlist.TwoDimensional = listSvr
		sort.Sort(sortlist)
		isvr, _ := m.svrPool.Load(listSvr[0][2])
		svr := isvr.(*registeredServer)
		m.addPickTimes(listSvr[0][2], svr)
		return svr.svrAddr, nil
	}
	return "", fmt.Errorf(`No matching server was found with the name %s`, svrname)
}

// ReportDeadServer 报告无法访问的服务，从缓存中删除
func (m *Etcdv3Client) ReportDeadServer(addr string) {
	m.svrPool.Range(func(k, v interface{}) bool {
		s := v.(*registeredServer)
		if s.svrAddr == addr {
			m.svrPool.Delete(k.(string))
			return false
		}
		return true
	})
}
