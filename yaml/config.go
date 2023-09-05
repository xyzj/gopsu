// Package yaml yaml格式的配置文件封装
package yaml

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config yaml 配置结构
type Config struct {
	name string // 配置文件名
}

// New 初始化
func New(f string) *Config {
	x, err := filepath.Abs(f)
	if err != nil {
		x = f
	}
	return &Config{
		name: x,
	}
}

// Write 写配置
func (c *Config) Write(d interface{}) error {
	b, err := yaml.Marshal(d)
	if err != nil {
		return err
	}
	return os.WriteFile(c.name, b, 0664)
}

// Read 读配置
func (c *Config) Read(d interface{}) error {
	b, err := os.ReadFile(c.name)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, d)
}

// Fullpath 配置文件完整路径
func (c *Config) Fullpath() string {
	return c.name
}
