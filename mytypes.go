package gopsu

import (
	"fmt"

	"github.com/xyzj/gopsu/json"
)

// SliceFlag 切片型参数，仅支持字符串格式
type SliceFlag []string

// String 返回参数
func (f *SliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*f))
}

// Set 设置值
func (f *SliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// PwdString 序列化反序列化时可自动加密解密字符串，用于敏感字段
type PwdString string

func (p *PwdString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*p = PwdString(DecodeString(s))
	return nil
}
func (p *PwdString) MarshalJSON() ([]byte, error) {
	if string(*p) == "" {
		return []byte("\"\""), nil
	}
	return []byte("\"" + CodeString(string(*p)) + "\""), nil
}
func (p *PwdString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*p = PwdString(DecodeString(s))
	return nil
}
func (p PwdString) MarshalYAML() (interface{}, error) {
	return CodeString(string(p)), nil
}
