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
	items sync.Map
	// items []*confItem
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
		c.Clean()
		var remarkbuf bytes.Buffer
		buf := bufio.NewReader(file)
		for {
			line, ex := buf.ReadString('\n')
			if ex != nil || io.EOF == ex {
				break
			}
			line = TrimString(line)
			if len(line) == 0 {
				remarkbuf.WriteString("\r\n")
				continue
			}
			if strings.Index(line, "#") > -1 && strings.Index(line, "#") < 5 {
				remarkbuf.WriteString(line)
				continue
			} else {
				s := strings.SplitN(line, "=", 2)
				if len(s[1]) > 0 {
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
	// for _, v := range c.items {
	// 	if v.key == key {
	// 		v.value = value
	// 		return true
	// 	}
	// }
	return found
}

// DelItem 删除配置项
func (c *ConfData) DelItem(key string) {
	c.items.Delete(key)
}

// SetItem 设置配置项
func (c *ConfData) SetItem(key, value, remark string) bool {
	// defer return false
	key = TrimString(key)
	value = TrimString(value)
	if strings.HasPrefix(remark, "#") == false {
		remark = fmt.Sprintf("#%s", TrimString(remark))
	}
	c.items.Store(key, &confItem{
		key:    key,
		value:  value,
		remark: remark,
	})
	// var found = false
	// for _, v := range c.items {
	// 	if v.key == key {
	// 		v.value = value
	// 		v.remark = remark
	// 		found = true
	// 		break
	// 	}
	// }
	// if found == false {
	// 	item := &confItem{
	// 		key:    key,
	// 		value:  value,
	// 		remark: remark,
	// 	}
	// 	c.items = append(c.items, item)
	// }
	return true
}

// GetItemDefault 获取配置项的value
func (c *ConfData) GetItemDefault(key, value string, remark ...string) string {
	v, err := c.GetItem(key)
	if err != nil {
		c.SetItem(key, value, remark[0])
		v = TrimString(value)
	}
	return v
	// if c.items == nil {
	// 	return ""
	// }
	// found := false
	// var x string
	// for _, v := range c.items {
	// 	if v.key == TrimString(key) {
	// 		x = v.value
	// 		found = true
	// 		break
	// 	}
	// }
	// if found == false {
	// 	if len(remark) > 0 {
	// 		c.SetItem(key, value, remark[0])
	// 	} else {
	// 		c.SetItem(key, value, "")
	// 	}
	// 	c.Save()
	// 	return value
	// }
	// return x
}

// GetItem 获取配置项的value
func (c *ConfData) GetItem(key string) (string, error) {
	v, ok := c.items.Load(key)
	if ok {
		return v.(*confItem).value, nil
	}
	return "", fmt.Errorf("Key does not exist")
	// if c.items == nil {
	// 	return "", fmt.Errorf("config data error")
	// }
	// found := false
	// var x string
	// for _, v := range c.items {
	// 	if v.key == TrimString(key) {
	// 		x = v.value
	// 		found = true
	// 		break
	// 	}
	// }
	// if found == false {
	// 	return "", errors.New("Key does not exist")
	// }
	// return x, nil
}

// GetKeys 获取所有配置项的key
func (c *ConfData) GetKeys() []string {
	var keys = make([]string, 0)
	c.items.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true
	})
	// keys := make([]string, len(c.items))
	// for k, v := range c.items {
	// 	keys[k] = TrimString(v.key)
	// }
	return keys
}

// Save 保存配置文件
func (c *ConfData) Save() error {
	file, ex := os.OpenFile(c.fileFullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if ex != nil {
		return ex
	}
	defer file.Close()
	var err error
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
	for _, v := range ss {
		remark := make([]string, 0)
		x := strings.Split(v.remark, "#")
		for _, v := range x {
			if TrimString(v) != "" {
				remark = append(remark, "#"+v)
			}
		}
		_, ex := file.WriteString(fmt.Sprintf("%s\r\n%s=%s\r\n\r\n", strings.Join(remark, "\r\n"), v.key, v.value))
		if ex != nil {
			return ex
		}
	}
	// for _, v := range c.items {
	// 	x := strings.Split(v.remark, "#")
	// 	remark := make([]string, 0)
	// 	for _, v := range x {
	// 		if len(v) > 0 {
	// 			remark = append(remark, "#"+v)
	// 		}
	// 	}
	// 	_, ex := file.WriteString(fmt.Sprintf("%s\r\n%s=%s\r\n\r\n", strings.Join(remark, "\r\n"), v.key, v.value))
	// 	if ex != nil {
	// 		return ex
	// 	}
	// }
	return err
}

// GetAll 获取所有配置项的key，value，以json字符串返回
func (c *ConfData) GetAll() string {
	var s = make([]string, 0, c.Len())
	c.items.Range(func(k, v interface{}) bool {
		s = append(s, fmt.Sprintf("\"%s\":\"%s\"", v.(*confItem).key, v.(*confItem).value))
		return true
	})
	// for _, v := range c.items {
	// 	s = append(s, fmt.Sprintf("\"%s\":\"%s\"", v.key, v.value))
	// }
	return fmt.Sprintf("{%s}", strings.Join(s, ","))
}

// Clean 清除所有配置项
func (c *ConfData) Clean() {
	c.items.Range(func(k, v interface{}) bool {
		c.items.Delete(k)
		return true
	})
	// c.items = make([]*confItem, 0)
}

// Len 获取配置数量
func (c *ConfData) Len() int {
	var i int
	c.items.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	return i
	// return len(c.items)
}

// LoadConfig load config file
func LoadConfig(fullpath string) (*ConfData, error) {
	c := &ConfData{
		fileFullPath: fullpath,
		fileName:     path.Base(fullpath),
		// items:        make([]*confItem, 0),
	}
	c.Reload()
	// if IsExist(fullpath) {
	// 	file, ex := os.Open(fullpath)
	// 	if ex != nil {
	// 		return c, ex
	// 	}
	// 	defer func() (*ConfData, error) {
	// 		if ex := recover(); ex != nil {
	// 			file.Close()
	// 		}
	// 		return c, fmt.Errorf("file format error")
	// 	}()
	// 	var remarkbuf bytes.Buffer
	// 	buf := bufio.NewReader(file)
	// 	for {
	// 		line, ex := buf.ReadString('\n')
	// 		if ex != nil || io.EOF == ex {
	// 			break
	// 		}
	// 		line = TrimString(line)
	// 		if len(line) == 0 {
	// 			remarkbuf.WriteString("\r\n")
	// 			continue
	// 		}
	// 		if strings.Index(line, "#") > -1 && strings.Index(line, "#") < 5 {
	// 			remarkbuf.WriteString(line)
	// 			continue
	// 		} else {
	// 			s := strings.SplitN(line, "=", 2)
	// 			if len(s[1]) > 0 {
	// 				c.SetItem(s[0], s[1], remarkbuf.String())
	// 			}
	// 			remarkbuf.Reset()
	// 		}
	// 	}
	// 	return c, nil
	// }
	ex := c.Save()
	return c, ex
}
