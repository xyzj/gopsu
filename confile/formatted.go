package config

import (
	"os"

	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/mapfx"
	"gopkg.in/yaml.v3"
)

type FormatType byte

const (
	YAML FormatType = iota
	JSON
)

// Formatted yaml/json 格式化配置文件
type Formatted[ITEM any] struct {
	items      *mapfx.StructMap[string, ITEM]
	filepath   string
	formatType FormatType
}

// NewFormatFile 创建一个新的自定义结构的yaml/json配置文件
func NewFormatFile[ITEM any](configfile string, ft FormatType) *Formatted[ITEM] {
	y := &Formatted[ITEM]{filepath: configfile, formatType: ft}
	y.FromFile("")
	return y
}

// PutItem 添加一个配置项
func (f *Formatted[ITEM]) PutItem(key string, value *ITEM) {
	f.items.Store(key, value)
}

// GetItem 读取一个配置项
func (f *Formatted[ITEM]) GetItem(key string) (*ITEM, bool) {
	return f.items.Load(key)
}

// Clone 获取所有配置
func (f *Formatted[ITEM]) Clone() map[string]*ITEM {
	return f.items.Clone()
}

// FromFile 从文件读取配置
func (f *Formatted[ITEM]) FromFile(configfile string) error {
	if configfile != "" {
		f.filepath = configfile
	}
	f.items = mapfx.NewStructMap[string, ITEM]()
	b, err := os.ReadFile(f.filepath)
	if err != nil {
		return err
	}
	switch f.formatType {
	case YAML:
		return f.fromYAML(b)
	case JSON:
		return f.fromJSON(b)
	}
	return nil
}

// ToFile 写入文件
func (f *Formatted[ITEM]) ToFile() error {
	switch f.formatType {
	case YAML:
		return f.toYAML()
	case JSON:
		return f.toJSON()
	}
	return nil
}

// fromYAML 从yaml文件读取
func (f *Formatted[ITEM]) fromYAML(b []byte) error {
	x := make(map[string]*ITEM)
	err := yaml.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	for k, v := range x {
		f.items.Store(k, v)
	}
	return nil
}

// fromJSON 从json文件读取
func (f *Formatted[ITEM]) fromJSON(b []byte) error {
	x := make(map[string]*ITEM)
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	for k, v := range x {
		f.items.Store(k, v)
	}
	return nil
}

// toYAML 写入yaml文件
func (f *Formatted[ITEM]) toYAML() error {
	b, err := yaml.Marshal(f.items.Clone())
	if err != nil {
		return err
	}
	return os.WriteFile(f.filepath, b, 0644)
}

// toJSON 写入json文件
func (f *Formatted[ITEM]) toJSON() error {
	b, err := json.Marshal(f.items.Clone())
	if err != nil {
		return err
	}
	return os.WriteFile(f.filepath, b, 0644)
}
