package gopsu

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
)

// 配置项结构体
type confItem struct {
	key    string
	value  string
	remark string
}

// ConfData 配置文件结构体
type ConfData struct {
	items        sync.Map
	fileFullPath string
	fileName     string
}

// Reload reload config file
func (c *ConfData) Reload() error {
	if IsExist(c.fileFullPath) {
		file, ex := os.Open(c.fileFullPath)
		if ex != nil {
			return ex
		}
		defer file.Close()
		c.Clear()
		var remarkbuf bytes.Buffer
		buf := bufio.NewReader(file)
		for {
			line, ex := buf.ReadString('\n')
			if ex != nil || io.EOF == ex {
				break
			}
			line = TrimString(line)
			if len(line) == 0 {
				// remarkbuf.WriteString("\r\n")
				continue
			}
			if strings.Contains(line, "#") && strings.Index(line, "#") < 5 {
				remarkbuf.WriteString(line)
				continue
			} else {
				s := strings.SplitN(line, "=", 2)
				if len(s) > 1 {
					c.SetItem(s[0], s[1], remarkbuf.String())
				}
				remarkbuf.Reset()
			}
		}
		return nil
	}
	return fmt.Errorf("file not found")
}

// UpdateItem 更新配置项
func (c *ConfData) UpdateItem(key, value string) bool {
	key = TrimString(key)
	value = TrimString(value)
	var found = false
	c.items.Range(func(k, v interface{}) bool {
		if k.(string) == key {
			v.(*confItem).value = value
			found = true
			return false
		}
		return true
	})
	return found
}

// DelItem 删除配置项
func (c *ConfData) DelItem(key string) {
	c.items.Delete(TrimString(key))
}

// SetItem 设置配置项
func (c *ConfData) SetItem(key, value, remark string) bool {
	// defer return false
	key = TrimString(key)
	value = TrimString(value)
	remark = TrimString(remark)
	if !strings.HasPrefix(remark, "#") {
		remark = fmt.Sprintf("#%s", remark)
	}
	c.items.Store(key, &confItem{
		key:    key,
		value:  value,
		remark: remark,
	})
	return true
}

// GetItemDefault 获取配置项的value
func (c *ConfData) GetItemDefault(key, value string, remark ...string) string {
	key = TrimString(key)
	value = TrimString(value)
	remarks := TrimString(strings.Join(remark, "#"))
	v, err := c.GetItem(key)
	if err != nil {
		c.SetItem(key, value, remarks)
		v = value
	}
	return v
}

// GetItem 获取配置项的value
func (c *ConfData) GetItem(key string) (string, error) {
	v, ok := c.items.Load(TrimString(key))
	if ok {
		return v.(*confItem).value, nil
	}
	return "", fmt.Errorf("key does not exist")
}

// GetItemDetail 获取配置项的value
func (c *ConfData) GetItemDetail(key string) (string, string, error) {
	v, ok := c.items.Load(TrimString(key))
	if ok {
		return v.(*confItem).value, v.(*confItem).remark, nil
	}
	return "", "", fmt.Errorf("key does not exist")
}

// GetKeys 获取所有配置项的key
func (c *ConfData) GetKeys() []string {
	var keys = make([]string, 0)
	c.items.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true
	})
	return keys
}

// Save 保存配置文件
func (c *ConfData) Save() error {
	var ss = make([]*confItem, c.Len())
	var i int
	c.items.Range(func(k, v interface{}) bool {
		ss[i] = v.(*confItem)
		i++
		return true
	})
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].key < ss[j].key
	})
	file, err := os.OpenFile(c.fileFullPath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, v := range ss {
		x := strings.Split(v.remark, "#")
		for _, v := range x {
			v = TrimString(v)
			if v != "" {
				file.WriteString(fmt.Sprintf("#%s\r\n", v))
			}
		}
		file.WriteString(fmt.Sprintf("%s=%s\r\n\r\n", v.key, v.value))
	}
	return nil
}

// GetAll 获取所有配置项的key，value，以json字符串返回
func (c *ConfData) GetAll() string {
	var s = make([]string, 0, c.Len())
	c.items.Range(func(k, v interface{}) bool {
		s = append(s, fmt.Sprintf("\"%s\":\"%s\"", v.(*confItem).key, v.(*confItem).value))
		return true
	})
	return fmt.Sprintf("{%s}", strings.Join(s, ","))
}

// Clear 清除所有配置项
func (c *ConfData) Clear() {
	c.items.Range(func(k, v interface{}) bool {
		c.items.Delete(k)
		return true
	})
}

// Len 获取配置数量
func (c *ConfData) Len() int {
	var i int
	c.items.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	return i
}

// FullPath 配置文件完整路径
func (c *ConfData) FullPath() string {
	return c.fileFullPath
}

// LoadConfig load config file
func LoadConfig(fullpath string) (*ConfData, error) {
	c := &ConfData{
		items:        sync.Map{},
		fileFullPath: fullpath,
		fileName:     path.Base(fullpath),
	}
	err := c.Reload()
	// if err != nil {
	// 	return c, err
	// }
	// err = c.Save()
	return c, err
}
