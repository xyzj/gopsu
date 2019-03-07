package microgo

import (
	"context"
	"crypto/tls"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"go.etcd.io/etcd/clientv3"
)

// Etcdv3Client 微服务结构体
type Etcdv3Client struct {
	etcdRoot      string           // etcd注册根路经
	etcdAddr      []string         // etcd服务地址
	etcdKATime    time.Duration    // 心跳间隔
	etcdKATimeout time.Duration    // 心跳超时
	etcdClient    *clientv3.Client // 连接实例
	svrName       string           // 服务名称
	svrPool       sync.Map         // 线程安全服务信息字典
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
func NewEtcdv3Client(etcdaddr []string, timeka, timekao time.Duration) (*Etcdv3Client, error) {
	return NewEtcdv3ClientTLS(etcdaddr, timeka, timekao, nil)
}

// NewEtcdv3ClientTLS 获取新的微服务结构（tls）
func NewEtcdv3ClientTLS(etcdaddr []string, timeka, timekao time.Duration, tlsconf *tls.Config) (*Etcdv3Client, error) {
	m := &Etcdv3Client{
		etcdRoot:      "wlst-micro",
		etcdAddr:      etcdaddr,
		etcdKATime:    timeka,
		etcdKATimeout: timekao,
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            m.etcdAddr,
		DialTimeout:          2 * time.Second,
		DialKeepAliveTime:    m.etcdKATime,
		DialKeepAliveTimeout: m.etcdKATimeout,
		TLS:                  tlsconf,
	})
	if err != nil {
		return nil, err
	}
	m.etcdClient = cli
	return m, nil
}

// listServers 查询根路径下所有服务
func (m *Etcdv3Client) listServers() error {
	defer func() error {
		if err := recover(); err != nil {
			fmt.Printf("%+v\n", err)
		}
		return nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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
	if r.svrPickTimes >= 0xffffffff {
		r.svrPickTimes = 0
	} else {
		r.svrPickTimes++
	}
	m.svrPool.Store(k, r)
}

// SetRoot 自定义根路径
func (m *Etcdv3Client) SetRoot(root string) {
	m.etcdRoot = root
}

// Register 服务注册
func (m *Etcdv3Client) Register(svrname, svrip, intfc, protoname string, svrport int) error {
	m.svrName = svrname
	js, _ := sjson.Set("", "ip", svrip)
	js, _ = sjson.Set(js, "port", svrport)
	js, _ = sjson.Set(js, "name", svrname)
	js, _ = sjson.Set(js, "INTFC", intfc)
	js, _ = sjson.Set(js, "protocol", protoname)
	js, _ = sjson.Set(js, "timeConnect", time.Now().Unix())
	js, _ = sjson.Set(js, "timeActive", time.Now().Unix())

	lease := clientv3.NewLease(m.etcdClient)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	lresp, _ := lease.Grant(ctx, int64(m.etcdKATimeout.Seconds()))
	_, err := m.etcdClient.Put(ctx, fmt.Sprintf("/%s/%s/%s-%s", m.etcdRoot, svrname, svrname, gopsu.GetUUID1()), js, clientv3.WithLease(lresp.ID))
	cancel()
	if err != nil {
		return err
	}
	go func() {
		ch, _ := lease.KeepAlive(context.TODO(), lresp.ID)
		t := time.NewTicker(time.Second * 2)
		for _ = range t.C {
			<-ch
		}
	}()
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
	default:
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

// Picker 服务选择
func (m *Etcdv3Client) Picker(svrname string, intfc ...string) (string, error) {
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

// ReportDeadServer 报告无法访问的服务
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
