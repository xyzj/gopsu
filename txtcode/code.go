package txtcode

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// GbkToUtf8 gbk编码转utf8
func GbkToUtf8(s []byte) ([]byte, error) {
	if utf8.Valid(s) {
		return s, nil
	}
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return s, e
	}
	return d, nil
}

// Utf8ToGbk utf8编码转gbk
func Utf8ToGbk(s []byte) ([]byte, error) {
	// if !isUtf8(s) {
	// 	return s, nil
	// }
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return s, e
	}
	return d, nil
}

// func isGBK(data []byte) bool {
// 	length := len(data)
// 	var i int = 0
// 	for i < length {
// 		if data[i] <= 0x7f {
// 			//编码0~127,只有一个字节的编码，兼容ASCII码
// 			i++
// 			continue
// 		} else {
// 			//大于127的使用双字节编码，落在gbk编码范围内的字符
// 			if data[i] >= 0x81 &&
// 				data[i] <= 0xfe &&
// 				data[i+1] >= 0x40 &&
// 				data[i+1] <= 0xfe &&
// 				data[i+1] != 0xf7 {
// 				i += 2
// 				continue
// 			} else {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// func preNUm(data byte) int {
// 	var mask byte = 0x80
// 	var num int = 0
// 	//8bit中首个0bit前有多少个1bits
// 	for i := 0; i < 8; i++ {
// 		if (data & mask) == mask {
// 			num++
// 			mask = mask >> 1
// 		} else {
// 			break
// 		}
// 	}
// 	return num
// }
// func isUtf8(data []byte) bool {
// 	i := 0
// 	for i < len(data) {
// 		if (data[i] & 0x80) == 0x00 {
// 			// 0XXX_XXXX
// 			i++
// 			continue
// 		} else if num := preNUm(data[i]); num > 2 {
// 			// 110X_XXXX 10XX_XXXX
// 			// 1110_XXXX 10XX_XXXX 10XX_XXXX
// 			// 1111_0XXX 10XX_XXXX 10XX_XXXX 10XX_XXXX
// 			// 1111_10XX 10XX_XXXX 10XX_XXXX 10XX_XXXX 10XX_XXXX
// 			// 1111_110X 10XX_XXXX 10XX_XXXX 10XX_XXXX 10XX_XXXX 10XX_XXXX
// 			// preNUm() 返回首个字节的8个bits中首个0bit前面1bit的个数，该数量也是该字符所使用的字节数
// 			i++
// 			for j := 0; j < num-1; j++ {
// 				//判断后面的 num - 1 个字节是不是都是10开头
// 				if (data[i] & 0xc0) != 0x80 {
// 					return false
// 				}
// 				i++
// 			}
// 		} else {
// 			//其他情况说明不是utf-8
// 			return false
// 		}
// 	}
// 	return true
// }

// EncodeUTF16BE 将字符串编码成utf16be的格式，用于cdma短信发送
func EncodeUTF16BE(s string) []byte {
	a := utf16.Encode([]rune(s))
	var b bytes.Buffer
	for _, v := range a {
		b.Write([]byte{byte(v >> 8), byte(v)})
	}
	return b.Bytes()
}

// String2Unicode 字符串转4位unicode编码
func String2Unicode(s string) string {
	var str string
	for _, v := range s {
		str += fmt.Sprintf("%04X", v)
	}
	return str
}

// SMSUnicode 编码短信
func SMSUnicode(s string) []string {
	return SplitStringWithLen(String2Unicode(s), 67*4)
}

// SplitStringWithLen 按指定长度分割字符串
//
//	s-原始字符串
//	l-切割长度
func SplitStringWithLen(s string, l int) []string {
	rs := []rune(s)
	var ss = make([]string, 0)
	xs := ""
	for k, v := range rs {
		xs = xs + string(v)
		if (k+1)%l == 0 {
			ss = append(ss, xs)
			xs = ""
		}
	}
	if len(xs) > 0 {
		ss = append(ss, xs)
	}
	return ss
}
