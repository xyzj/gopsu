package yaml

import (
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/xyzj/gopsu"
	yaml "gopkg.in/yaml.v3"
)

// Config yaml 配置结构
type Config struct {
	locker    sync.RWMutex
	name      string // 配置文件名
	overwrite bool   // 是否允许覆盖已有key
	items     map[string]interface{}
}

// New 初始化
// 默认允许覆盖现有值
func New() *Config {
	return &Config{
		overwrite: true,
		items:     make(map[string]interface{}),
	}
}

// SetFilepath 设置配置文件路径
//  filepath： 文件路径
//  createnew：文件不存在时是否创建空文件
func (c *Config) SetFilepath(filepath string, createnew bool) error {
	if !gopsu.IsExist(filepath) {
		if !createnew {
			return fmt.Errorf("yaml file not found")
		}
		ioutil.WriteFile(filepath, []byte{}, 0664)
	}
	c.name = filepath
	return nil
}

// SetOverwrite 设置是否允许覆盖已有值
func (c *Config) SetOverwrite(b bool) {
	c.overwrite = b
}

// Read 读取配置
func (c *Config) Read() error {
	c.locker.Lock()
	defer c.locker.Unlock()
	b, err := ioutil.ReadFile(c.name)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, c.items)
}

// Write 写入配置
func (c *Config) Write() error {
	c.locker.RLock()
	defer c.locker.RUnlock()
	b, err := yaml.Marshal(c.items)
	if err != nil {
		return err
	}
	ioutil.WriteFile(c.name, b, 0664)
	return nil
}

// Set 设置配置值
func (c *Config) Set(key string, value interface{}) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	if _, ok := c.items[key]; ok && !c.overwrite {
		return fmt.Errorf("key: " + key + " already exist")
	}
	c.items[key] = value
	return nil
}

// Get 获取某个值
func (c *Config) Get(key string) (interface{}, bool) {
	c.locker.RLock()
	defer c.locker.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

// Del 删除某个值
func (c *Config) Del(key string) {
	c.locker.Lock()
	defer c.locker.Unlock()
	delete(c.items, key)
}

// Clear 清空所有值
func (c *Config) Clear() {
	c.locker.Lock()
	defer c.locker.Unlock()
	c.items = make(map[string]interface{})
}

// Range 遍历所有值
func (c *Config) Range(f func(key string, value interface{}) bool) {
	c.locker.RLock()
	defer c.locker.RUnlock()
	for k, v := range c.items {
		if !f(k, v) {
			break
		}
	}
}
