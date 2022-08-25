package main

import (
	"fmt"
	"log"
	"time"
)

func aaa(a, b, c string, d, e int) {
	println(fmt.Sprintf("%s, %s, %s, --- %d %d", a, b, c, d, e))
}

func main() {
	log.SetFlags(log.Ltime)
	log.Println("---- main")
	time.Sleep(time.Second * 3)
}
