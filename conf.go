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

	"github.com/xyzj/gopsu/mapfx"
)

// 配置项结构体
type item struct {
	Key   string
	Value string `json:"value"`
	Tip   string `json:"tip"`
}

// ConfData 配置文件结构体
type ConfData struct {
	items        *mapfx.StructMap[string, item]
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
			c.items.Store(strings.Split(s, "=")[0], &item{Key: strings.Split(s, "=")[0], Value: strings.Split(s, "=")[1], Tip: tip})
			tip = ""
		}
	}
	return nil
}

// AddOrUpdate 更新配置项
func (c *ConfData) AddOrUpdate(key, value, tip string) {
	c.items.Store(key, &item{
		Key:   key,
		Value: value,
		Tip:   tip,
	})
}

// DelItem 删除配置项
func (c *ConfData) DelItem(key string) {
	c.items.Delete(key)
}

// UpdateItem 更新配置项
func (c *ConfData) UpdateItem(key, value string) bool {
	x, ok := c.items.Load(key)
	if !ok {
		x = &item{Key: key, Value: value}
	}
	x.Value = value
	c.items.Store(key, x)
	return true
}

// SetItem 设置配置项
func (c *ConfData) SetItem(key, value, tip string) bool {
	if !strings.HasPrefix(tip, "#") {
		tip = "# " + tip
	}
	c.items.Store(key, &item{Key: key, Value: value, Tip: tip})
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
	v, ok := c.items.Load(key)
	if ok {
		return v.Value, nil
	}

	return "", errors.New("key is not found")
}

// GetKeys 获取所有配置项的key
func (c *ConfData) GetKeys() []string {
	var keys = make([]string, 0)
	x := c.items.Clone()
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
	var ss = make([]*item, 0, c.items.Len())
	x := c.items.Clone()
	for _, v := range x {
		ss = append(ss, v)
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
	x := c.items.Clone()
	buf := make([]string, 0)
	for k, v := range x {
		buf = append(buf, "\""+k+"\":\""+v.Value+"\"")
	}
	return "{" + strings.Join(buf, ",") + "}"
}

// Clear 清除所有配置项
func (c *ConfData) Clear() {
	c.items.Clean()
}

// Len 获取配置数量
func (c *ConfData) Len() int {
	return c.items.Len()
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
		items:        mapfx.NewStructMap[string, item](),
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
