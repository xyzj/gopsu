package microgo

import (
	"context"
	"crypto/tls"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	clientv3 "go.etcd.io/etcd/clientv3"
)

const (
	leaseTimeout   = 7
	contextTimeout = 3 * time.Second
)

// Etcdv3Client 微服务结构体
type Etcdv3Client struct {
	// etcdLog      *io.Writer       // 日志
	// etcdLogLevel int              // 日志等级
	etcdRoot   string           // etcd注册根路经
	etcdAddr   []string         // etcd服务地址
	etcdClient *clientv3.Client // 连接实例
	svrName    string           // 服务名称
	svrPool    sync.Map         // 线程安全服务信息字典
	svrDetail  string           // 服务信息
	logger     gopsu.Logger     //  日志接口
	realIP     string           // 所在电脑ip
	etcdKey    string
}

// RegisteredServer 获取到的服务注册信息
type registeredServer struct {
	svrName       string // 服务名称
	svrAddr       string // 服务地址
	svrPickTimes  int    // 命中次数
	svrProtocol   string // 服务使用数据格式
	svrInterface  string // 服务发布的接口类型
	svrActiveTime int64  // 服务查询时间
	svrKey        string // 服务注册key
	svrRealIP     string
}

func (rs *registeredServer) addPickTimes() {
	if rs.svrPickTimes >= 0xffffff {
		rs.svrPickTimes = 0
	} else {
		rs.svrPickTimes++
	}
}

func (rs *registeredServer) updateActive() {
	rs.svrActiveTime = time.Now().Unix()
}

func (rs *registeredServer) expired(now int64) bool {
	return now-rs.svrActiveTime >= 5
}

// NewEtcdv3Client 获取新的微服务结构
func NewEtcdv3Client(etcdaddr []string, username, password string) (*Etcdv3Client, error) {
	return NewEtcdv3ClientTLS(etcdaddr, "", "", "", username, password)
}

// NewEtcdv3ClientTLS 获取新的微服务结构（tls）
func NewEtcdv3ClientTLS(etcdaddr []string, certfile, keyfile, cafile, username, password string) (*Etcdv3Client, error) {
	m := &Etcdv3Client{
		etcdRoot: "wlst-micro",
		etcdAddr: etcdaddr,
		logger:   &gopsu.NilLogger{},
	}
	m.realIP = gopsu.RealIP(false)
	var tlsconf *tls.Config
	var err error
	if gopsu.IsExist(certfile) && gopsu.IsExist(keyfile) && gopsu.IsExist(cafile) {
		tlsconf, err = gopsu.GetClientTLSConfig(certfile, keyfile, cafile)
		if err != nil {
			return nil, err
		}
	} else {
		tlsconf = nil
	}
	cf := clientv3.Config{
		Endpoints:   m.etcdAddr,
		DialTimeout: 2 * time.Second,
		TLS:         tlsconf,
	}
	if username != "" && password != "" {
		cf.Username = username
		cf.Password = password
	}
	m.etcdClient, err = clientv3.New(cf)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// listServers 查询根路径下所有服务
func (m *Etcdv3Client) listServers() error {
	defer func() error {
		if err := recover(); err != nil {
			m.logger.Error("etcd list error: " + errors.WithStack(err.(error)).Error())
			return err.(error)
		}
		return nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	resp, err := m.etcdClient.Get(ctx, fmt.Sprintf("/%s/", m.etcdRoot), clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}
	// 先清理
	// m.svrPool.Range(func(key interface{}, value interface{}) bool {
	// 	m.svrPool.Delete(key)
	// 	return true
	// })
	// 重新添加
	for _, v := range resp.Kvs {
		va := gjson.ParseBytes(v.Value)
		if !va.Exists() {
			continue
		}
		if ss, ok := m.svrPool.Load(string(v.Key)); !ok { // 未记录该服务
			s := &registeredServer{
				svrName:       va.Get("name").String(),
				svrAddr:       fmt.Sprintf("%s:%s", va.Get("ip").String(), va.Get("port").String()),
				svrPickTimes:  0,
				svrProtocol:   va.Get("protocol").String(),
				svrInterface:  va.Get("INTFC").String(),
				svrActiveTime: time.Now().Unix(),
				svrKey:        gopsu.String(v.Key),
				svrRealIP:     va.Get("source").String(),
			}
			if s.svrName == "" {
				x := strings.Split(s.svrKey, "/")
				if len(x) > 2 {
					s.svrName = x[1]
				}
			}
			m.svrPool.Store(s.svrKey, s)
		} else {
			ss.(*registeredServer).updateActive()
		}
		// a, ok := m.svrPool.LoadOrStore(s.svrKey, s)
		// if ok {
		// 	s := a.(*registeredServer)
		// 	s.svrActiveTime = time.Now().Unix()
		// 	m.svrPool.Store(s.svrKey, s)
		// }
	}
	return nil
}

// AllServices 返回所有注册服务的信息
func (m *Etcdv3Client) AllServices() string {
	var s string
	var t = time.Now().Unix()
	m.svrPool.Range(func(key interface{}, value interface{}) bool {
		if value.(*registeredServer).expired(t) {
			m.svrPool.Delete(key)
			return true
		}
		s, _ = sjson.Set(s, key.(string), []string{value.(*registeredServer).svrInterface + "://" + value.(*registeredServer).svrAddr, value.(*registeredServer).svrRealIP})
		return true
	})
	return s
}

// addPickTimes 增加计数器
// func (m *Etcdv3Client) addPickTimes(k string, r *registeredServer) {
// 	if r.svrPickTimes >= 0xffffff { // 防止溢出
// 		r.svrPickTimes = 0
// 	} else {
// 		r.svrPickTimes++
// 	}
// 	m.svrPool.Store(k, r)
// }

// SetRoot 自定义根路径
//
// args:
//  root: 注册根路径，默认'wlst-micro'
func (m *Etcdv3Client) SetRoot(root string) {
	m.etcdRoot = root
}

// SetLogger 设置日志记录器
func (m *Etcdv3Client) SetLogger(l gopsu.Logger) {
	m.logger = l
	// m.etcdLog = l
	// m.etcdLogLevel = level
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
func (m *Etcdv3Client) Register(svrname, svrip, svrport, intfc, protoname string) error {
	m.svrName = svrname
	m.etcdKey = fmt.Sprintf("/%s/%s/%s_%s", m.etcdRoot, m.svrName, m.svrName, gopsu.GetUUID1())
	if svrip == "" {
		svrip = gopsu.RealIP(false)
	}
	js, _ := sjson.Set("", "ip", svrip)
	js, _ = sjson.Set(js, "port", svrport)
	js, _ = sjson.Set(js, "name", svrname)
	js, _ = sjson.Set(js, "INTFC", intfc)
	js, _ = sjson.Set(js, "protocol", protoname)
	js, _ = sjson.Set(js, "timeConnect", time.Now().Unix())
	js, _ = sjson.Set(js, "timeActive", time.Now().Unix())
	js, _ = sjson.Set(js, "source", m.realIP)
	m.svrDetail = js

	// 监视线程，在etcd崩溃并重启时重新注册
	// 注册
	var err error
	var leaseGrantResp *clientv3.LeaseGrantResponse
	var lease clientv3.Lease
	var leaseid clientv3.LeaseID
	var keepRespChan <-chan *clientv3.LeaseKeepAliveResponse
	// RUN:
	if m.etcdClient.ActiveConnection() == nil {
		return fmt.Errorf("connection not active")
	}
	m.listServers()
	lease = clientv3.NewLease(m.etcdClient)
	if leaseGrantResp, err = lease.Grant(context.Background(), leaseTimeout); err != nil {
		m.logger.Error(fmt.Sprintf("Create lease error: %s", err.Error()))
		return fmt.Errorf("create lease error: %s", err.Error())
	}
	leaseid = leaseGrantResp.ID
	_, err = m.etcdClient.Put(context.Background(), m.etcdKey, m.svrDetail, clientv3.WithLease(leaseid))
	if err != nil {
		m.logger.Error(fmt.Sprintf("Registration to %s failed: %v", m.etcdAddr, err.Error()))
		return fmt.Errorf("registration to %s failed: %v", m.etcdAddr, err.Error())
	}
	m.logger.System(fmt.Sprintf("Registration to %v as `%s://%s:%s/%s` success.", m.etcdAddr, intfc, svrip, svrport, svrname))
	keepRespChan, err = lease.KeepAlive(context.Background(), leaseid)
	if err != nil {
		m.logger.Error(fmt.Sprintf("Keep lease error: %s", err.Error()))
		return fmt.Errorf("keep lease error: %s", err.Error())
	}
	// func() {
	// 	defer func() { recover() }()
	t := time.NewTicker(time.Second * 3)
	for {
		select {
		case keepResp := <-keepRespChan:
			if keepResp == nil {
				m.logger.Error("Lease failure, try to reboot.")
				return fmt.Errorf("lease failure, try to reboot")
			}
		case <-t.C:
			m.listServers()
		}
	}
	// }()
	// time.Sleep(time.Duration(rand.Intn(2000)+1500) * time.Millisecond)
	// goto RUN
	return nil
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
			for range time.After(time.Second * 3) {
				if m.etcdClient.ActiveConnection() == nil {
					return
				}

				m.listServers()
			}
		}()
	}
	return nil
}
func (m *Etcdv3Client) pickerList(svrname string, intfc ...string) []*registeredServer {
	t := time.Now().Unix()
	listSvr := make([]*registeredServer, 0)
	m.svrPool.Range(func(k, v interface{}) bool {
		s := v.(*registeredServer)
		// 删除无效服务信息
		if s.expired(t) {
			m.svrPool.Delete(k)
			return true
		}
		if s.svrName == svrname {
			listSvr = append(listSvr, s)
		}
		if len(listSvr) >= 4 {
			return false
		}
		return true
	})
	return listSvr
}

// func (m *Etcdv3Client) pickerList(svrname string, intfc ...string) [][]string {
// 	t := time.Now().Unix()
// 	listSvr := make([][]string, 0)
// 	// 找到所有同名服务
// 	switch len(intfc) {
// 	case 1: // 匹配服务名称和接口类型
// 		m.svrPool.Range(func(k, v interface{}) bool {
// 			s := v.(*registeredServer)
// 			// 删除无效服务信息
// 			if t-s.svrActiveTime >= 5 {
// 				m.svrPool.Delete(k)
// 				return true
// 			}
// 			if s.svrName == svrname && s.svrInterface == intfc[0] {
// 				listSvr = append(listSvr, []string{fmt.Sprintf("%012d", s.svrPickTimes), s.svrAddr, k.(string)})
// 			}
// 			return true
// 		})
// 	case 2: // 匹配服务名称，接口类型，和协议类型
// 		m.svrPool.Range(func(k, v interface{}) bool {
// 			s := v.(*registeredServer)
// 			// 删除无效服务信息
// 			if t-s.svrActiveTime >= 5 {
// 				m.svrPool.Delete(k)
// 				return true
// 			}
// 			if s.svrName == svrname && s.svrInterface == intfc[0] && s.svrProtocol == intfc[1] {
// 				listSvr = append(listSvr, []string{fmt.Sprintf("%012d", s.svrPickTimes), s.svrAddr, k.(string)})
// 			}
// 			return true
// 		})
// 	default: // 仅匹配服务名称
// 		m.svrPool.Range(func(k, v interface{}) bool {
// 			s := v.(*registeredServer)
// 			// 删除无效服务信息
// 			if t-s.svrActiveTime >= 5 {
// 				m.svrPool.Delete(k)
// 				return true
// 			}
// 			if s.svrName == svrname {
// 				listSvr = append(listSvr, []string{fmt.Sprintf("%012d", s.svrPickTimes), s.svrAddr, k.(string)})
// 			}
// 			return true
// 		})
// 	}
// 	return listSvr
// }

// PickerAll 服务选择
//
// args:
//  svrname: 服务名称
//  intfc: 服务类型，协议类型
// return:
//  string: 服务地址
//  error
func (m *Etcdv3Client) PickerAll(svrname string, intfc ...string) []string {
	listSvr := m.pickerList(svrname)
	var allSvr = make([]string, 0)
	for _, v := range listSvr {
		allSvr = append(allSvr, v.svrAddr)
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
	listSvr := m.pickerList(svrname)
	if len(listSvr) > 0 {
		// 排序，获取命中最少的服务地址
		sort.Slice(listSvr, func(i int, j int) bool {
			return listSvr[i].svrPickTimes < listSvr[j].svrPickTimes
		})
		listSvr[0].addPickTimes()
		// sortlist := &gopsu.StringSliceSort{}
		// sortlist.TwoDimensional = listSvr
		// sort.Sort(sortlist)
		// isvr, _ := m.svrPool.Load(listSvr[0][2])
		// svr := isvr.(*registeredServer)
		// m.addPickTimes(listSvr[0][2], svr)
		return listSvr[0].svrAddr, nil
	}
	return "", fmt.Errorf(`no matching server was found with the name %s`, svrname)
}

// PickerDetail 服务选择,如果是http服务，同时返回协议头如http(s)://ip:port
//
// args:
//  svrname: 服务名称
//  intfc: 服务类型，协议类型
// return:
//  string: 服务地址
//  error
func (m *Etcdv3Client) PickerDetail(svrname string, intfc ...string) (string, error) {
	listSvr := m.pickerList(svrname)
	if len(listSvr) > 0 {
		// 排序，获取命中最少的服务地址
		sort.Slice(listSvr, func(i int, j int) bool {
			return listSvr[i].svrPickTimes < listSvr[j].svrPickTimes
		})
		listSvr[0].addPickTimes()
		if strings.HasPrefix(listSvr[0].svrInterface, "http") {
			return listSvr[0].svrInterface + "://" + listSvr[0].svrAddr, nil
		}
		return listSvr[0].svrAddr, nil
		// sortlist := &gopsu.StringSliceSort{}
		// sortlist.TwoDimensional = listSvr
		// sort.Sort(sortlist)
		// isvr, _ := m.svrPool.Load(listSvr[0][2])
		// svr := isvr.(*registeredServer)
		// m.addPickTimes(listSvr[0][2], svr)
		// if strings.HasPrefix(svr.svrInterface, "http") {
		// 	return svr.svrInterface + "://" + svr.svrAddr, nil
		// }
		// return svr.svrAddr, nil
	}
	return "", fmt.Errorf(`no matching server was found with the name %s`, svrname)
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
