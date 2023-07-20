package gopsu

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/mohae/deepcopy"
)

// 配置项结构体
type item struct {
	Key   string
	Value string `json:"value"`
	Tip   string `json:"tip"`
}
type mapItem struct {
	locker sync.RWMutex
	data   map[string]*item
}

func (m *mapItem) clean() {
	m.locker.Lock()
	m.data = make(map[string]*item)
	m.locker.Unlock()
}
func (m *mapItem) store(key, value, tip string) {
	m.locker.Lock()
	m.data[key] = &item{
		Key:   key,
		Value: value,
		Tip:   tip,
	}
	m.locker.Unlock()
}
func (m *mapItem) update(key, value string) bool {
	m.locker.Lock()
	v, ok := m.data[key]
	m.locker.Unlock()
	if !ok {
		return false
	}
	m.store(key, value, v.Tip)
	return true
}
func (m *mapItem) delete(key string) {
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}
func (m *mapItem) load(key string) (string, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if !ok || v == nil {
		return "", false
	}
	return v.Value, true
}
func (m *mapItem) copy() map[string]*item {
	m.locker.RLock()
	v := deepcopy.Copy(m.data).(map[string]*item)
	m.locker.RUnlock()
	return v
}
func (m *mapItem) len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}

// ConfData 配置文件结构体
type ConfData struct {
	items        *mapItem
	fileFullPath string
	fileName     string
}

// Reload reload config file
func (c *ConfData) Reload() error {
	b, err := os.ReadFile(c.fileFullPath)
	if err != nil {
		return err
	}
	ss := strings.Split(String(b), "\n")
	tip := ""
	for _, v := range ss {
		s := strings.TrimSpace(v)
		if strings.HasPrefix(s, "#") {
			tip += s
			continue
		}
		if strings.Contains(s, "=") {
			c.items.store(strings.Split(s, "=")[0], strings.Split(s, "=")[1], tip)
			tip = ""
		}
	}
	return nil

	// if IsExist(c.fileFullPath) {
	// 	file, ex := os.Open(c.fileFullPath)
	// 	if ex != nil {
	// 		return ex
	// 	}
	// 	defer file.Close()
	// 	c.Clear()
	// 	var remarkbuf bytes.Buffer
	// 	buf := bufio.NewReader(file)
	// 	for {
	// 		line, ex := buf.ReadString('\n')
	// 		if ex != nil || io.EOF == ex {
	// 			break
	// 		}
	// 		line = strings.TrimSpace(line)
	// 		if len(line) == 0 {
	// 			// remarkbuf.WriteString("\r\n")
	// 			continue
	// 		}
	// 		if strings.Contains(line, "#") && strings.Index(line, "#") < 5 {
	// 			remarkbuf.WriteString(line)
	// 			continue
	// 		} else {
	// 			s := strings.SplitN(line, "=", 2)
	// 			if len(s) > 1 {
	// 				c.SetItem(s[0], s[1], remarkbuf.String())
	// 			}
	// 			remarkbuf.Reset()
	// 		}
	// 	}
	// 	return nil
	// }
	// return fmt.Errorf("file not found")
}

// AddOrUpdate 更新配置项
func (c *ConfData) AddOrUpdate(key, value, tip string) {
	c.items.store(key, value, tip)
}

// DelItem 删除配置项
func (c *ConfData) DelItem(key string) {
	c.items.delete(key)
}

// UpdateItem 更新配置项
func (c *ConfData) UpdateItem(key, value string) bool {
	return c.items.update(key, value)
}

// SetItem 设置配置项
func (c *ConfData) SetItem(key, value, tip string) bool {
	if !strings.HasPrefix(tip, "#") {
		tip = "# " + tip
	}
	c.items.store(key, value, tip)
	return true
}

// GetItemDefault 获取配置项的value
func (c *ConfData) GetItemDefault(key, value string, remark ...string) string {
	remarks := strings.Join(remark, "# ")
	v, err := c.GetItem(key)
	if err != nil {
		c.SetItem(key, value, remarks)
		v = value
	}
	return v
}

// GetItem 获取配置项的value
func (c *ConfData) GetItem(key string) (string, error) {
	v, ok := c.items.load(key)
	if ok {
		return v, nil
	}

	return "", errors.New("key is not found")
}

// GetKeys 获取所有配置项的key
func (c *ConfData) GetKeys() []string {
	var keys = make([]string, 0)
	x := c.items.copy()
	for k := range x {
		keys = append(keys, k)
	}
	return keys
}

// Save 保存配置文件
func (c *ConfData) Save() error {
	if c.fileFullPath == "" {
		return fmt.Errorf("no file specified")
	}
	var ss = make([]*item, c.items.len())
	var i int
	x := c.items.copy()
	for _, v := range x {
		ss[i] = v
		i++
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Key < ss[j].Key
	})
	buf := bytes.Buffer{}
	for _, v := range ss {
		if len(v.Tip) > 0 {
			for _, tip := range strings.Split(v.Tip, "#") {
				s := strings.TrimSpace(tip)
				if s == "" {
					continue
				}
				buf.WriteString("# " + s + "\r\n")
			}
		}
		buf.WriteString(v.Key + "=" + v.Value + "\r\n\r\n")
	}
	return os.WriteFile(c.fileFullPath, buf.Bytes(), 0666)
}

// GetAll 获取所有配置项的key，value，以json字符串返回
func (c *ConfData) GetAll() string {
	x := c.items.copy()
	buf := make([]string, 0)
	for k, v := range x {
		buf = append(buf, "\""+k+"\":\""+v.Value+"\"")
	}
	return "{" + strings.Join(buf, ",") + "}"
}

// Clear 清除所有配置项
func (c *ConfData) Clear() {
	c.items.clean()
}

// Len 获取配置数量
func (c *ConfData) Len() int {
	return c.items.len()
}

// FullPath 配置文件完整路径
func (c *ConfData) FullPath() string {
	return c.fileFullPath
}

// SetFullPath 设置配置文件目录
func (c *ConfData) SetFullPath(p string) {
	c.fileFullPath = p
}

// LoadConfig load config file
func LoadConfig(fullpath string) (*ConfData, error) {
	c := &ConfData{
		items: &mapItem{
			locker: sync.RWMutex{},
			data:   make(map[string]*item),
		},
		fileFullPath: fullpath,
		fileName:     path.Base(fullpath),
	}
	if fullpath == "" {
		return c, nil
	}
	dir := filepath.Dir(fullpath)
	if !IsExist(dir) {
		os.MkdirAll(dir, 0775)
	}
	err := c.Reload()
	// if err != nil {
	// 	return c, err
	// }
	// err = c.Save()
	return c, err
}
