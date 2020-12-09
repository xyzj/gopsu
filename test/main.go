package main

import (
	"fmt"

	"github.com/buger/jsonparser"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	var b = make([]byte, 0)
	b = []byte("{}")
	var err error
	for i := 0; i < 10; i++ {
		for j := 0; j < 5; j++ {
			b, err = jsonparser.Set(b, []byte(fmt.Sprintf("%d", 123*i+i)), fmt.Sprintf("key%06d", i), fmt.Sprintf("[%d]", j),"abc")
			if err != nil {
				println("---" + err.Error())
			}
		}
	}
	// for i := 0; i < 10; i++ {
	// 	for j := 0; j < 10; j++ {
	// 		b, err = jsonparser.Set(b, []byte("asldfkalsdfjlasjdflajflsjdfklaj9102830217123jo1hfsahdfalkfd"), fmt.Sprintf("key%06d", i), fmt.Sprintf("subkey%03d", j))
	// 		if err != nil {
	// 			println(err.Error())
	// 		}
	// 	}
	// }
	println(string(b))
}
