package yaml

import (
	"io/ioutil"

	"github.com/goccy/go-yaml"
)

// Config yaml 配置结构
type Config struct {
	name string // 配置文件名
}

// New 初始化
func New(f string) *Config {
	return &Config{
		name: f,
	}
}

// Write 写配置
func (c *Config) Write(d interface{}) error {
	b, err := yaml.Marshal(d)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.name, b, 0664)
}

// Read 读配置
func (c *Config) Read(d interface{}) error {
	b, err := ioutil.ReadFile(c.name)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, d)
}
