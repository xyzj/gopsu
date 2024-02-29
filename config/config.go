// Package config 类ini的配置文件库，支持注释信息
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/mapfx"
	"gopkg.in/yaml.v3"
)

// File 配置文件
type File struct {
	items      *mapfx.StructMap[string, Item]
	data       *bytes.Buffer
	filepath   string
	formatType FormatType
}

// Item 配置内容，包含注释，key,value,是否加密value
type Item struct {
	Key          string  `json:"-" yaml:"-"`
	Value        VString `json:"value" yaml:"value"`
	Comment      string  `json:"comment" yaml:"comment"`
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

// NewConfig 创建一个key:value格式的配置文件
//
//	依据文件的扩展名，支持yaml和json格式的文件
func NewConfig(filepath string) *File {
	f := &File{}
	f.FromFile(filepath)
	return f
}

// Keys 获取所有Key
func (f *File) Keys() []string {
	return f.items.Keys()
	// ss := make([]string, 0, f.items.Len())
	// f.items.ForEach(func(key string, value *Item) bool {
	// 	ss = append(ss, key)
	// 	return true
	// })
	// return ss
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

// ForEach 遍历所有值
func (f *File) ForEach(do func(key string, value VString) bool) {
	f.items.ForEach(func(key string, value *Item) bool {
		return do(key, value.Value)
	})
}

// Len 获取配置数量
func (f *File) Len() int {
	return f.items.Len()
}

// Has 判断key是否存在
func (f *File) Has(key string) bool {
	return f.items.Has(key)
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

// GetAll 返回所有配置项
func (f *File) GetAll() string {
	x := f.items.Clone()
	buf := make([]string, 0)
	for k, v := range x {
		buf = append(buf, "\""+k+"\":\""+v.Value.String()+"\"")
	}
	return "{" + strings.Join(buf, ",") + "}"
}

// FromFile 从文件载入配置
func (f *File) FromFile(configfile string) error {
	if configfile == "" {
		return nil
	}
	if configfile != "" {
		f.filepath = configfile
	}
	if f.data == nil {
		f.data = &bytes.Buffer{}
	} else {
		f.data.Reset()
	}
	if f.items == nil {
		f.items = mapfx.NewStructMap[string, Item]()
	} else {
		f.items.Clean()
	}
	b, err := os.ReadFile(f.filepath)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	f.data.Write(b)
	if b[0] == '{' {
		if f.fromJSON(b) == nil {
			return nil
		}
	}
	if f.fromYAML(b) == nil {
		return nil
	}
	ss := strings.Split(f.data.String(), "\n")
	tip := make([]string, 0)
	for _, v := range ss {
		s := strings.TrimSpace(v)
		if strings.HasPrefix(s, "#") {
			if xt := strings.TrimSpace(s[1:]); xt != "" {
				tip = append(tip, xt)
			}
			continue
		}
		it := strings.Split(s, "=")
		if len(it) != 2 {
			continue
		}
		f.items.Store(it[0], &Item{Key: it[0], Value: VString(it[1]), Comment: strings.Join(tip, "\n")})
		tip = []string{}
	}
	return nil
}

// Save 将配置写入文件，依据文件扩展名判断写入格式
func (f *File) Save() error {
	return f.ToFile()
}

// ToFile 将配置写入文件，依据文件扩展名判断写入格式
func (f *File) ToFile() error {
	switch strings.ToLower(filepath.Ext(f.filepath)) {
	case ".yaml":
		return f.ToYAML()
	case ".json":
		return f.ToJSON()
	}
	f.Print()
	return os.WriteFile(f.filepath, f.data.Bytes(), 0644)
}

// ToYAML 保存为yaml格式文件
func (f *File) ToYAML() error {
	b, err := yaml.Marshal(f.items.Clone())
	if err != nil {
		return err
	}
	return os.WriteFile(f.filepath, b, 0644)
}

// ToJSON 保存为json格式文件
func (f *File) ToJSON() error {
	b, err := json.MarshalIndent(f.items.Clone(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.filepath, b, 0644)
}

func (f *File) fromYAML(b []byte) error {
	x := make(map[string]*Item)
	err := yaml.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	for k, v := range x {
		f.items.Store(k, &Item{Key: k, Value: v.Value, Comment: v.Comment})
	}
	return nil
}

func (f *File) fromJSON(b []byte) error {
	x := make(map[string]*Item)
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	for k, v := range x {
		f.items.Store(k, &Item{Key: k, Value: v.Value, Comment: v.Comment})
	}
	return nil
}
