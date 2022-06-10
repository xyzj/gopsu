package pinyin

import (
	"strings"
)

// Meta
const (
	Version   = "0.19.0"
	Author    = "mozillazg, 闲耘"
	License   = "MIT"
	Copyright = "Copyright (c) 2016 mozillazg, 闲耘"
)

// XPinyin 返回拼音
//  s: 字符串
//  fl: 是否只返回首字母
func XPinyin(s string, fl bool) string {
	pys := make([]string, 0)
	for _, r := range s {
		value, ok := PinyinDict[int(r)]
		if ok {
			if fl { // 首字母
				pys = append(pys, string(strings.Split(value, ",")[0][0]))
				continue
			}
			pys = append(pys, strings.Split(value, ",")[0])
		} else {
			pys = append(pys, string(r))
		}
	}
	// if fl {
	return strings.Join(pys, "")
	// }
	// return strings.Join(pys, ",")
}

// XPinyinMatch 拼音全拼和首字母匹配
func XPinyinMatch(s, substr string) bool {
	if substr == "" {
		return true
	}
	// a, b := XPinyin(s)
	return strings.Contains(XPinyin(s, true), substr) || strings.Contains(XPinyin(s, false), substr)
}
