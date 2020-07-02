package gopsu

import (
	"sync"
	"time"
)

// XCache 可设置超时的缓存字典
type XCache struct {
	m   map[interface{}]*xCacheData
	len int
	// chanSet      chan *xCacheData
	chanSetTimeo chan *xCacheData
	chanReq      chan int
	chanGet      chan interface{}
	chanResp     chan interface{}
}

// xCacheData 可设置超时的缓存字典数据结构
type xCacheData struct {
	key    interface{}
	value  interface{}
	expire int64
}

// NewCache 创建新的缓存字典
//	l：字典大小,0-不限制
func NewCache(l int) *XCache {
	xc := &XCache{
		m:   make(map[interface{}]*xCacheData, l),
		len: l,
		// chanSet:      make(chan *xCacheData),
		chanSetTimeo: make(chan *xCacheData),
		chanReq:      make(chan int),
		chanGet:      make(chan interface{}),
		chanResp:     make(chan interface{}),
	}
	go xc.run()
	return xc
}

// Set 设置缓存数据
//	k: key
//	v: value
//	expire: 超时时间（ms）,0-不超时
func (xc *XCache) Set(k, v interface{}, expire int64) bool {
	return xc.SetWithHold(k, v, expire, 13)
}

// SetWithHold 设置缓存数据
//	k: key
//	v: value
//	expire: 超时时间（ms）,0-不超时
//	timeout: 写入超时，ms，0-始终等待
func (xc *XCache) SetWithHold(k, v interface{}, expire, timeout int64) bool {
	if expire <= 0 {
		expire = 316224000000
	}
	if timeout <= 0 {
		timeout = 316224000000
	}
	t := time.NewTicker(time.Millisecond * time.Duration(timeout))
	for {
		select {
		case <-t.C:
			return false
		case <-time.After(time.Millisecond * 7):
			xc.chanSetTimeo <- &xCacheData{
				key:    k,
				value:  v,
				expire: time.Now().UnixNano()/1000000 + expire,
			}
			b := <-xc.chanResp
			if b.(bool) == true {
				return true
			}
		}
	}
}

// Get 读取缓存数据
func (xc *XCache) Get(k interface{}) (interface{}, bool) {
	xc.chanGet <- k
	v := <-xc.chanResp
	return v, true
}

// Clear 清空缓存
func (xc *XCache) Clear() {
	xc.chanReq <- 1
}

// Len 获取缓存数量
func (xc *XCache) Len() int {
	xc.chanReq <- 0
	l := <-xc.chanResp
	return l.(int)
}

func (xc *XCache) run() {
	var exlocker sync.WaitGroup
	var t = time.NewTicker(time.Millisecond * 10)
RUN:
	exlocker.Add(1)
	go func() {
		defer func() {
			recover()
			exlocker.Done()
		}()
		for {
			select {
			// case set := <-xc.chanSet:
			// 	xc.m[set.key] = set
			case set := <-xc.chanSetTimeo:
				if len(xc.m) < xc.len {
					xc.m[set.key] = set
					xc.chanResp <- true
				} else {
					xc.chanResp <- false
				}
			case a := <-xc.chanReq:
				switch a {
				case 0: // 获取长度
					xc.chanResp <- len(xc.m)
				case 1: // 清空
					xc.m = make(map[interface{}]*xCacheData, xc.len)
				}
			case key := <-xc.chanGet:
				xc.chanResp <- xc.m[key]
			case <-t.C:
				tt := time.Now().UnixNano() / 1000000
				for k, v := range xc.m {
					if v.expire <= tt { // 过期
						delete(xc.m, k)
					}
				}
			}
		}
	}()
	time.Sleep(time.Second)
	exlocker.Wait()
	goto RUN
}
