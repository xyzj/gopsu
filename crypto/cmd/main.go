package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/xyzj/gopsu/crypto"
)

var (
	conf = flag.String("config", "config.json", "config file for dns and ip")
	cry  = flag.String("crypto", "ecc256", "crypto type, ecc256, ecc384, rsa2048, rsa4096")
)

type jsonConf struct {
	DNS []string `json:"dns"`
	IP  []string `json:"ip"`
}

func main() {
	flag.Parsed()
	for _, v := range os.Args {
		if v == "-h" || v == "--help" || v == "-help" {
			flag.PrintDefaults()
			return
		}
	}
	b, err := os.ReadFile(*conf)
	if err != nil {
		b = []byte("{}")
	}
	var cf = &jsonConf{
		DNS: []string{},
		IP:  []string{},
	}
	err = json.Unmarshal(b, cf)
	if err != nil {
		println("load config file error: " + err.Error())
		return
	}
	switch *cry {
	case "ecc384":
		ec := crypto.NewECC()
		_, _, err := ec.GenerateKey(crypto.ECSecp384r1)
		if err != nil {
			println("gen ecc-key error: " + err.Error())
			return
		}
		err = ec.CreateCert(cf.DNS, cf.IP)
		if err != nil {
			println("create ecc-cert error: " + err.Error())
			return
		}
	case "rsa2048":
		ec := crypto.NewRSA()
		_, _, err := ec.GenerateKey(crypto.RSA2048)
		if err != nil {
			println("gen rsa-key error: " + err.Error())
			return
		}
		err = ec.CreateCert(cf.DNS, cf.IP)
		if err != nil {
			println("create rsa-cert error: " + err.Error())
			return
		}
	case "rsa4096":
		ec := crypto.NewRSA()
		_, _, err := ec.GenerateKey(crypto.RSA4096)
		if err != nil {
			println("gen rsa-key error: " + err.Error())
			return
		}
		err = ec.CreateCert(cf.DNS, cf.IP)
		if err != nil {
			println("create rsa-cert error: " + err.Error())
			return
		}
	default:
		ec := crypto.NewECC()
		_, _, err := ec.GenerateKey(crypto.ECPrime256v1)
		if err != nil {
			println("gen ecc-key error: " + err.Error())
			return
		}
		err = ec.CreateCert(cf.DNS, cf.IP)
		if err != nil {
			println("create ecc-cert error: " + err.Error())
			return
		}
	}
	println("create cert file done.")
}
