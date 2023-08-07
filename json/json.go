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
	"unsafe"

	json "github.com/goccy/go-json"
	// "github.com/bytedance/sonic"
)

var (
	// json = sonic.Config{
	// 	// NoNullSliceOrMap:     true,
	// 	NoQuoteTextMarshaler: true,
	// }.Froze()

	// Marshal is exported by gin/json package.
	Marshal = xMarshal
	// Unmarshal is exported by gin/json package.
	Unmarshal = xUnmarshal
	// MarshalIndent is exported by gin/json package.
	MarshalIndent = json.MarshalIndent
	// MarshalToString return string and error
	MarshalToString = xMarshalToString
	// UnmarshalFromString get data from string
	UnmarshalFromString = xUnmarshalFromString
	// NewDecoder is exported by gin/json package.
	NewDecoder = json.NewDecoder
	// NewEncoder is exported by gin/json package.
	NewEncoder = json.NewEncoder
	// Valid check if valid json string
	Valid = json.Valid
)

// xMarshal json.MarshalWithOption
func xMarshal(v interface{}) ([]byte, error) {
	// return json.Marshal(v)
	return json.MarshalWithOption(v, json.DisableHTMLEscape(), json.UnorderedMap())
}

// xMarshalToString json.MarshalWithOption and return string
func xMarshalToString(v interface{}) (string, error) {
	b, err := xMarshal(v)
	if err == nil {
		return ToString(b), nil
	}
	return "", err
}

// xUnmarshal json.UnmarshalWithOption
func xUnmarshal(data []byte, v interface{}) error {
	// return json.Unmarshal(data, v)
	return json.UnmarshalNoEscape(data, v, json.DecodeFieldPriorityFirstWin())
}

// xUnmarshalFromString json.UnmarshalFromString
func xUnmarshalFromString(data string, v interface{}) error {
	return xUnmarshal(ToBytes(data), v)
}

// ToBytes 内存地址转换string
func ToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// ToString 内存地址转换[]byte
func ToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
