package yaml

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/mohae/deepcopy"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/tools"
)

type item struct {
	Tip   string `yaml:"tip" json:"tip"`
	Value string `yaml:"value" json:"value"`
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
		Value: value,
		Tip:   tip,
	}
	m.locker.Unlock()
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

// File yaml格式的配置文件
type File struct {
	filename string
	dir      string
	items    *mapItem
}

// AddOrUpdate 添加或更新值
func (c *File) AddOrUpdate(key, value, tip string) {
	c.items.store(key, value, tip)
}

// Load 读取一个值
func (c *File) Load(key string) (string, bool) {
	return c.items.load(key)
}

// LoadDefault 读取一个值，不存在时，使用默认值返回，并添加该键值
func (c *File) LoadDefault(key, value, tip string) string {
	v, ok := c.items.load(key)
	if !ok {
		c.items.store(key, value, tip)
		v = value
	}
	return v
}

// WriteFile 保存配置到文件
func (c *File) WriteFile() error {
	if !tools.IsExist(c.dir) {
		os.MkdirAll(c.dir, 0775)
	}
	b, err := yaml.Marshal(c.items.data)
	if err != nil {
		return err
	}
	return os.WriteFile(c.filename, b, 0664)
}

// ReadFile 读取配置文件
func (c *File) ReadFile() error {
	c.items.clean()
	b, err := os.ReadFile(c.filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &c.items.data)
	if err != nil {
		return err
	}
	return nil
}

// Show 显示全部配置
func (c *File) Show() string {
	b, err := json.Marshal(c.items.copy())
	if err != nil {
		return ""
	}
	return tools.String(b)
}

// NewFile 创建一个新配置文件
func NewFile(filename string) *File {
	filename, _ = filepath.Abs(filename)
	return &File{
		filename: filename,
		dir:      filepath.Dir(filename),
		items: &mapItem{
			locker: sync.RWMutex{},
			data:   make(map[string]*item),
		},
	}
}
