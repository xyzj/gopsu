// Copyright 2017 Bo-Yi Wu.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

//go:build !go_json && !(avx && (linux || windows || darwin) && amd64)

/*
Package json ： 使用 github.com/bytedance/sonic 替代 encoding/json，性能更好
*/
package json

import (
	"github.com/bytedance/sonic"
)

var (
	// 自定义配置 // sonic.ConfigFastest
	json = sonic.Config{
		NoNullSliceOrMap:        true,
		NoQuoteTextMarshaler:    true,
		NoValidateJSONMarshaler: true,
	}.Froze()
	// Valid 验证
	Valid = json.Valid
	// Marshal 序列化
	Marshal = json.Marshal
	// Unmarshal 反序列化
	Unmarshal = json.Unmarshal
	// MarshalIndent 带缩进的序列化
	MarshalIndent = json.MarshalIndent
	// MarshalToString 序列化为字符串
	MarshalToString = json.MarshalToString
	// UnmarshalFromString 从字符串反序列化
	UnmarshalFromString = json.UnmarshalFromString
)
