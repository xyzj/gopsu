// Copyright 2017 Bo-Yi Wu.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

//go:build !jsoniter && !go_json
// +build !jsoniter,!go_json

/*
Package json ： 使用 github.com/bytedance/sonic 替代 encoding/json，性能更好
*/
package json

import (
	jsonstd "encoding/json"
	"unsafe"

	json "github.com/goccy/go-json"
	// "github.com/bytedance/sonic"
)

var (
//	json = sonic.Config{
//		// NoNullSliceOrMap:     true,
//		NoQuoteTextMarshaler: true,
//	}.Froze()
)

// Valid json.Valid
func Valid(data []byte) bool {
	return json.Valid(data)
}

// Marshal json.MarshalWithOption
func Marshal(v interface{}) ([]byte, error) {
	return json.MarshalNoEscape(v)
	// return json.MarshalWithOption(v, json.UnorderedMap())
}

// MarshalIndent json.MarshalIndent
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// MarshalToString json.MarshalWithOption and return string
func MarshalToString(v interface{}) (string, error) {
	b, err := Marshal(v)
	if err == nil {
		return String(b), nil
	}
	return "", err
}

// Unmarshal json.UnmarshalWithOption
func Unmarshal(data []byte, v interface{}) error {
	err := json.UnmarshalNoEscape(data, v)
	if err != nil {
		return jsonstd.Unmarshal(data, v)
	}
	return nil
}

// UnmarshalFromString json.UnmarshalFromString
func UnmarshalFromString(data string, v interface{}) error {
	return Unmarshal(Bytes(data), v)
}

// Bytes 内存地址转换string
func Bytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// String 内存地址转换[]byte
func String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// PB2Json pb2格式转换为json []byte格式
func PB2Json(pb interface{}) []byte {
	jsonBytes, err := Marshal(pb)
	if err != nil {
		return nil
	}
	return jsonBytes
}

// PB2String pb2格式转换为json 字符串格式
func PB2String(pb interface{}) string {
	b, err := MarshalToString(pb)
	if err != nil {
		return ""
	}
	return b
}

// JSON2PB json字符串转pb2格式
func JSON2PB(js string, pb interface{}) error {
	err := Unmarshal(Bytes(js), &pb)
	return err
}
