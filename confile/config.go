// Package config 类ini的配置文件库，支持注释信息
package config

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/mapfx"
)

// VString value string, can parse to bool int64 float64
type VString string

func (rs VString) String() string {
	return string(rs)
}
func (rs VString) Bytes() []byte {
	return []byte(rs)
}
func (rs VString) TryBool() bool {
	v, _ := strconv.ParseBool(string(rs))
	return v
}
func (rs VString) TryInt64() int64 {
	return gopsu.String2Int64(string(rs), 10)
}
func (rs VString) TryFloat64() float64 {
	return gopsu.String2Float64(string(rs))
}
func (rs VString) TryDecode() string {
	if s := gopsu.DecodeString(string(rs)); s != "" {
		return s
	}
	return string(rs)
}

// Item 配置内容，包含注释，key,value,是否加密value
type Item struct {
	Key          string  `json:"key,omitempty" yaml:"key,omitempty"`
	Value        VString `json:"value,omitempty" yaml:"value,omitempty"`
	Comment      string  `json:"comment,omitempty" yaml:"comment,omitempty"`
	EncryptValue bool    `json:"-" yaml:"-"`
}

// String 把配置项格式化成字符串
func (i *Item) String() string {
	ss := strings.Split(i.Comment, "\n")
	xcom := ""
	for _, v := range ss {
		if strings.HasPrefix(v, "#") {
			xcom = v + "\n"
		} else {
			xcom = "# " + v + "\n"
		}
	}
	return fmt.Sprintf("\n%s%s=%s\n", xcom, i.Key, i.Value)
}

// NewConfig 创建一个新的配置文件
func NewConfig(filepath string) *File {
	f := &File{}
	f.FromFile(filepath)
	return f
}

// File 配置文件结构
type File struct {
	items    *mapfx.StructMap[string, Item]
	data     *bytes.Buffer
	filepath string
}

// DelItem 删除配置项
func (f *File) DelItem(key string) {
	f.items.Delete(key)
}

// PutItem 添加配置项
func (f *File) PutItem(item *Item) {
	if item.EncryptValue {
		item.Value = VString(gopsu.CodeString(item.Value.String()))
	}
	if v, ok := f.items.Load(item.Key); ok {
		if item.Comment == "" {
			item.Comment = v.Comment
		}
	}
	f.items.Store(item.Key, item)
}

// GetDefault 读取一个配置，若不存在，则添加这个配置
func (f *File) GetDefault(item *Item) VString {
	if v, ok := f.items.Load(item.Key); ok {
		return v.Value
	}
	f.PutItem(item)
	return item.Value
}

// GetItem 获取一个配置值
func (f *File) GetItem(key string) VString {
	if v, ok := f.items.Load(key); ok {
		return v.Value
	}
	return ""
}

// Len 获取配置数量
func (f *File) Len() int {
	return f.items.Len()
}

// Print 返回所有配置项
func (f *File) Print() string {
	var x = make([]*Item, 0, f.items.Len())
	f.items.ForEach(func(key string, value *Item) bool {
		x = append(x, value)
		return true
	})
	sort.Slice(x, func(i, j int) bool {
		return x[i].Key < x[j].Key
	})
	f.data.Reset()
	for _, v := range x {
		f.data.WriteString(v.String())
	}
	return f.data.String()
}

// FromFile 从文件载入配置
func (f *File) FromFile(configfile string) error {
	if configfile != "" {
		f.filepath = configfile
	}
	f.data = &bytes.Buffer{}
	f.items = mapfx.NewStructMap[string, Item]()
	b, err := os.ReadFile(f.filepath)
	if err != nil {
		return err
	}
	f.data.Write(b)
	ss := strings.Split(f.data.String(), "\n")
	tip := ""
	for _, v := range ss {
		s := strings.TrimSpace(v)
		if strings.HasPrefix(s, "#") {
			tip += s
			continue
		}
		it := strings.Split(s, "=")
		if len(it) != 2 {
			continue
		}
		f.items.Store(it[0], &Item{Key: it[0], Value: VString(it[1]), Comment: tip})
		tip = ""
	}
	return nil
}

// ToFile 将配置写入文件
func (f *File) ToFile() error {
	f.Print()
	return os.WriteFile(f.filepath, f.data.Bytes(), 0644)
}
