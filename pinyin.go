package gopsu

import (
	"regexp"
	"strings"
)

// Meta
// const (
// 	Version   = "0.14.0"
// 	Author    = "mozillazg, 闲耘"
// 	License   = "MIT"
// 	Copyright = "Copyright (c) 2016 mozillazg, 闲耘"
// )

// 拼音风格(推荐)
const (
	Normal      = 0 // 普通风格，不带声调（默认风格）。如： zhong guo
	Tone        = 1 // 声调风格1，拼音声调在韵母第一个字母上。如： zhōng guó
	Tone2       = 2 // 声调风格2，即拼音声调在各个韵母之后，用数字 [1-4] 进行表示。如： zho1ng guo2
	Tone3       = 8 // 声调风格3，即拼音声调在各个拼音之后，用数字 [1-4] 进行表示。如： zhong1 guo2
	Initials    = 3 // 声母风格，只返回各个拼音的声母部分。如： zh g
	FirstLetter = 4 // 首字母风格，只返回拼音的首字母部分。如： z g
	Finals      = 5 // 韵母风格，只返回各个拼音的韵母部分，不带声调。如： ong uo
	FinalsTone  = 6 // 韵母风格1，带声调，声调在韵母第一个字母上。如： ōng uó
	FinalsTone2 = 7 // 韵母风格2，带声调，声调在各个韵母之后，用数字 [1-4] 进行表示。如： o1ng uo2
	FinalsTone3 = 9 // 韵母风格3，带声调，声调在各个拼音之后，用数字 [1-4] 进行表示。如： ong1 uo2
)

var (
	// 声母表
	initialArray = strings.Split(
		"b,p,m,f,d,t,n,l,g,k,h,j,q,x,r,zh,ch,sh,z,c,s",
		",",
	)

	// 带音标字符。
	phoneticSymbol = map[string]string{
		"ā": "a1",
		"á": "a2",
		"ǎ": "a3",
		"à": "a4",
		"ē": "e1",
		"é": "e2",
		"ě": "e3",
		"è": "e4",
		"ō": "o1",
		"ó": "o2",
		"ǒ": "o3",
		"ò": "o4",
		"ī": "i1",
		"í": "i2",
		"ǐ": "i3",
		"ì": "i4",
		"ū": "u1",
		"ú": "u2",
		"ǔ": "u3",
		"ù": "u4",
		"ü": "v",
		"ǘ": "v2",
		"ǚ": "v3",
		"ǜ": "v4",
		"ń": "n2",
		"ň": "n3",
		"ǹ": "n4",
		"ḿ": "m2",
	}

	// 匹配带声调字符的正则表达式
	rePhoneticSymbol = regexp.MustCompile("[" + rePhoneticSymbolSource + "]")

	// 所有带声调的字符
	rePhoneticSymbolSource = func(m map[string]string) string {
		s := ""
		for k := range m {
			s = s + k
		}
		return s
	}(phoneticSymbol)

	// 匹配使用数字标识声调的字符的正则表达式
	reTone2 = regexp.MustCompile("([aeoiuvnm])([1-4])$")

	// 匹配 Tone2 中标识韵母声调的正则表达式
	reTone3 = regexp.MustCompile("^([a-z]+)([1-4])([a-z]*)$")

	finalExceptionsMap = map[string]string{
		"ū": "ǖ",
		"ú": "ǘ",
		"ǔ": "ǚ",
		"ù": "ǜ",
	}
	reFinalExceptions  = regexp.MustCompile(`^(j|q|x)(ū|ú|ǔ|ù)$`)
	reFinal2Exceptions = regexp.MustCompile(`^(j|q|x)u(\d?)$`)
)

// Pinyin 拼音转汉字
type Pinyin struct {
	Hanzi       string // 字符串
	Style       int    // 拼音风格（默认： Normal)
	Heteronym   bool   // 是否启用多音字模式（默认：禁用）
	Separator   string // join中使用的分隔符（默认："")
	IgnoreOther bool   // 是否忽略非汉字字符
}

// String 返回拼音字符串，多音字取第一个值
func (p *Pinyin) String() string {
	p.Heteronym = false
	pys := []string{}
	for _, r := range p.Hanzi {
		py := p.singlePinyin(r)
		if len(py) > 0 {
			pys = append(pys, p.singlePinyin(r)[0])
		}
	}
	return strings.Join(pys, p.Separator)
}

// Result 返回拼音列表，可设置支持多音字
func (p *Pinyin) Result() [][]string {
	pys := [][]string{}
	for _, r := range p.Hanzi {
		py := p.singlePinyin(r)
		if len(py) > 0 {
			pys = append(pys, p.singlePinyin(r))
		}
	}
	return pys
}

// singlePinyin 把单个 `rune` 类型的汉字转换为拼音.
func (p *Pinyin) singlePinyin(r rune) []string {
	value, ok := pinyinDict[int(r)]
	pys := []string{}
	if ok {
		pys = strings.Split(value, ",")

		if !p.Heteronym {
			pys = pys[:1]
		}
		pys = p.applyStyle(pys)
	} else {
		if !p.IgnoreOther {
			pys = []string{string(r)}
		}
	}
	return pys
}

// 获取单个拼音中的声母
func (p *Pinyin) initial(s string) string {
	for _, v := range initialArray {
		if strings.HasPrefix(s, v) {
			return v
		}
	}
	return ""
}

// 获取单个拼音中的韵母
func (p *Pinyin) final(s string) string {
	n := p.initial(s)
	if n == "" {
		return p.handleYW(s)
	}

	// 特例 j/q/x
	matches := reFinalExceptions.FindStringSubmatch(s)
	// jū -> jǖ
	if len(matches) == 3 && matches[1] != "" && matches[2] != "" {
		return finalExceptionsMap[matches[2]]
	}
	// ju -> jv, ju1 -> jv1
	s = reFinal2Exceptions.ReplaceAllString(s, "${1}v$2")
	return strings.Join(strings.SplitN(s, n, 2), "")
}

// 处理 y, w
func (p *Pinyin) handleYW(s string) string {
	// 特例 y/w
	if strings.HasPrefix(s, "yu") {
		s = "v" + s[2:] // yu -> v
	} else if strings.HasPrefix(s, "yi") {
		s = s[1:] // yi -> i
	} else if strings.HasPrefix(s, "y") {
		s = "i" + s[1:] // y -> i
	} else if strings.HasPrefix(s, "wu") {
		s = s[1:] // wu -> u
	} else if strings.HasPrefix(s, "w") {
		s = "u" + s[1:] // w -> u
	}
	return s
}

func (p *Pinyin) toFixed(s string) string {
	if p.Style == Initials {
		return p.initial(s)
	}
	origP := s

	// 替换拼音中的带声调字符
	py := rePhoneticSymbol.ReplaceAllStringFunc(s, func(m string) string {
		symbol := phoneticSymbol[m]
		switch p.Style {
		// 不包含声调
		case Normal, FirstLetter, Finals:
			// 去掉声调: a1 -> a
			m = reTone2.ReplaceAllString(symbol, "$1")
		case Tone2, FinalsTone2, Tone3, FinalsTone3:
			// 返回使用数字标识声调的字符
			m = symbol
		default:
			// 声调在头上
		}
		return m
	})

	switch p.Style {
	// 将声调移动到最后
	case Tone3, FinalsTone3:
		py = reTone3.ReplaceAllString(py, "$1$3$2")
	}
	switch p.Style {
	// 首字母
	case FirstLetter:
		py = py[:1]
	// 韵母
	case Finals, FinalsTone, FinalsTone2, FinalsTone3:
		// 转换为 []rune unicode 编码用于获取第一个拼音字符
		// 因为 string 是 utf-8 编码不方便获取第一个拼音字符
		rs := []rune(origP)
		switch string(rs[0]) {
		// 因为鼻音没有声母所以不需要去掉声母部分
		case "ḿ", "ń", "ň", "ǹ":
		default:
			py = p.final(py)
		}
	}
	return py
}

func (p *Pinyin) applyStyle(s []string) []string {
	newP := []string{}
	for _, v := range s {
		newP = append(newP, p.toFixed(v))
	}
	return newP
}
