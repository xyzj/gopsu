package gopsu

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// 配置项结构体
type confItem struct {
	key    string
	value  string
	remark string
}

// 配置文件结构体
type ConfData struct {
	items        []*confItem
	fileFullPath string
	fileName     string
}

// SetItem 设置配置项
func (c *ConfData) SetItem(key, value, remark string) bool {
	// defer return false
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if strings.HasPrefix(remark, "#") == false {
		remark = fmt.Sprintf("#%s", strings.TrimSpace(remark))
	}
	found := false
	for _, v := range c.items {
		if v.key == key {
			v.value = value
			v.remark = remark
			found = true
			break
		}
	}
	if found == false {
		item := &confItem{
			key:    key,
			value:  value,
			remark: remark,
		}
		c.items = append(c.items, item)
	}
	return true
}

// GetItemDefault 获取配置项的value
func (c *ConfData) GetItemDefault(key, value string, remark ...string) string {
	found := false
	var x string
	for _, v := range c.items {
		if v.key == strings.TrimSpace(key) {
			x = v.value
			found = true
			break
		}
	}
	if found == false {
		if len(remark) > 0 {
			c.SetItem(key, value, remark[0])
		} else {
			c.SetItem(key, value, "")
		}
		c.Save()
		return value
	}
	return x
}

// GetItem 获取配置项的value
func (c *ConfData) GetItem(key string) (string, error) {
	found := false
	var x string
	for _, v := range c.items {
		if v.key == strings.TrimSpace(key) {
			x = v.value
			found = true
			break
		}
	}
	if found == false {
		return "", errors.New("Key does not exist")
	}
	return x, nil
}

// GetKeys 获取所有配置项的key
func (c *ConfData) GetKeys() []string {
	keys := make([]string, len(c.items))
	for k, v := range c.items {
		keys[k] = strings.TrimSpace(v.key)
	}
	return keys
}

// Save 保存配置文件
func (c *ConfData) Save() error {
	file, ex := os.OpenFile(c.fileFullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if ex != nil {
		return ex
	}
	defer file.Close()
	for _, v := range c.items {
		x := strings.Split(v.remark, "#")
		remark := make([]string, 0)
		for _, v := range x {
			if len(v) > 0 {
				remark = append(remark, "#"+v)
			}
		}
		_, ex := file.WriteString(fmt.Sprintf("%s\r\n%s=%s\r\n\r\n", strings.Join(remark, "\r\n"), v.key, v.value))
		if ex != nil {
			return ex
		}
	}
	return nil
}

// GetAll 获取所有配置项的key，value，以json字符串返回
func (c *ConfData) GetAll() string {
	var s = make([]string, 0, len(c.items))
	for _, v := range c.items {
		s = append(s, fmt.Sprintf("\"%s\":\"%s\"", v.key, v.value))
	}
	return fmt.Sprintf("{%s}", strings.Join(s, ","))
}

// Len 获取配置数量
func (c *ConfData) Len() int {
	return len(c.items)
}

// LoadConfig load config file
func LoadConfig(fullpath string) (*ConfData, error) {
	c := &ConfData{
		fileFullPath: fullpath,
		fileName:     path.Base(fullpath),
		items:        make([]*confItem, 0),
	}
	if IsExist(fullpath) {
		file, ex := os.Open(fullpath)
		if ex != nil {
			return nil, ex
		}
		defer func() (*ConfData, error) {
			if ex := recover(); ex != nil {
				file.Close()
			}
			return nil, errors.New("file format error.")
		}()
		var remarkbuf bytes.Buffer
		buf := bufio.NewReader(file)
		for {
			line, ex := buf.ReadString('\n')
			if ex != nil || io.EOF == ex {
				break
			}
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				remarkbuf.WriteString("\r\n")
				continue
			}
			if strings.HasPrefix(line, "#") {
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
		return c, nil
	}
	ex := c.Save()
	return c, ex
}
