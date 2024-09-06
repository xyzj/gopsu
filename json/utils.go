package json

import "unsafe"

// Bytes 内存地址转换string
func Bytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// String 内存地址转换[]byte
func String(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
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
