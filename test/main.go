package main

import (
	"github.com/xyzj/gopsu"
)

func main() {
	appkey := "jbq-cszmgls-zgypt-wFF327nyQbAz"
	appsecret := "VpD6198L95gUhA4JbCT9HF58Xka0j95Zq8N3IgGJ"
	hmacworker := gopsu.GetNewCryptoWorker(gopsu.CryptoHMACSHA256)
	hmacworker.SetKey(appsecret, "")
	hmacworker.Encrypt("拼接的数据")
}
