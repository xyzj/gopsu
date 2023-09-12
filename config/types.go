package config

import (
	"strconv"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/json"
)

// VString value string, can parse to bool int64 float64
type VString string

// String reutrn string
func (rs VString) String() string {
	return string(rs)
}

// Bytes reutrn []byte
func (rs VString) Bytes() []byte {
	return []byte(rs)
}

// TryBool reutrn bool
func (rs VString) TryBool() bool {
	v, _ := strconv.ParseBool(string(rs))
	return v
}

// TryInt64 reutrn int64
func (rs VString) TryInt64() int64 {
	return gopsu.String2Int64(string(rs), 10)
}

// TryFloat64 reutrn fl
func (rs VString) TryFloat64() float64 {
	return gopsu.String2Float64(string(rs))
}

// TryDecode try decode the value, if failed, return the origin
func (rs VString) TryDecode() string {
	if s := gopsu.DecodeString(string(rs)); s != "" {
		return s
	}
	return string(rs)
}

// PwdString 序列化反序列化时可自动加密解密字符串，用于敏感字段
type PwdString string

func (p *PwdString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*p = PwdString(gopsu.DecodeString(s))
	return nil
}
func (p *PwdString) MarshalJSON() ([]byte, error) {
	if string(*p) == "" {
		return []byte("\"\""), nil
	}
	return []byte("\"" + gopsu.CodeString(string(*p)) + "\""), nil
}
func (p *PwdString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*p = PwdString(gopsu.DecodeString(s))
	return nil
}
func (p PwdString) MarshalYAML() (interface{}, error) {
	if string(p) == "" {
		return "", nil
	}
	return gopsu.CodeString(string(p)), nil
}
