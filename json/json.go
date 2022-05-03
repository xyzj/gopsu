// Copyright 2017 Bo-Yi Wu.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

//go:build !jsoniter && !go_json
// +build !jsoniter,!go_json

package json

import (
	"unsafe"

	json "github.com/goccy/go-json"
)

var (
	// Marshal is exported by gin/json package.
	Marshal = XMarshal
	// Unmarshal is exported by gin/json package.
	Unmarshal = XUnmarshal
	// MarshalIndent is exported by gin/json package.
	MarshalIndent = json.MarshalIndent
	// MarshalToString return string and error
	MarshalToString = XMarshalToString
	// UnmarshalFromString get data from string
	UnmarshalFromString = XUnmarshalFromString
	// NewDecoder is exported by gin/json package.
	NewDecoder = json.NewDecoder
	// NewEncoder is exported by gin/json package.
	NewEncoder = json.NewEncoder
	// Valid check if valid json string
	Valid = json.Valid
)

// XMarshal json.MarshalWithOption
func XMarshal(v interface{}) ([]byte, error) {
	return json.MarshalWithOption(v, json.DisableHTMLEscape(), json.UnorderedMap())
}

// XMarshalToString json.MarshalWithOption and return string
func XMarshalToString(v interface{}) (string, error) {
	b, err := json.MarshalWithOption(v, json.DisableHTMLEscape(), json.UnorderedMap())
	if err == nil {
		return *(*string)(unsafe.Pointer(&b)), nil
	}
	return "", err
}

// XUnmarshal json.UnmarshalWithOption
func XUnmarshal(data []byte, v interface{}) error {
	return json.UnmarshalNoEscape(data, v, json.DecodeFieldPriorityFirstWin())
}

// XUnmarshalFromString json.UnmarshalFromString
func XUnmarshalFromString(data string, v interface{}) error {
	return json.UnmarshalNoEscape(toBytes(data), v, json.DecodeFieldPriorityFirstWin())
}

// Bytes 内存地址转换string
func toBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
