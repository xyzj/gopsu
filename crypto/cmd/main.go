package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/xyzj/gopsu/crypto"
)

var (
	conf   = flag.String("config", "config.json", "config file for dns and ip")
	cry    = flag.String("crypto", "ecc256", "crypto type, ecc256, ecc384, rsa2048, rsa4096, sm2")
	sample = flag.Bool("sample", false, "create a sample config file")
)

func main() {
	flag.Parse()
	for _, v := range os.Args {
		if v == "-h" || v == "--help" || v == "-help" {
			flag.PrintDefaults()
			return
		}
	}
	if *sample {
		cf := &crypto.CertOpt{
			DNS:     []string{"localhost"},
			IP:      []string{"127.0.0.1"},
			RootKey: "root-key.ec.pem",
			RootCa:  "root.ec.pem",
		}
		b, err := json.MarshalIndent(cf, "", "  ")
		if err != nil {
			println(err.Error())
			return
		}
		os.WriteFile("config.json", b, 0o664)
		println("create sample config file done")
		return
	}
	b, err := os.ReadFile(*conf)
	if err != nil {
		b = []byte("{}")
	}
	cf := &crypto.CertOpt{
		DNS: []string{},
		IP:  []string{},
	}
	err = json.Unmarshal(b, cf)
	if err != nil {
		println("load config file error: " + err.Error())
		return
	}
	switch *cry {
	case "sm2":
		ec := crypto.NewSM2()
		err = ec.CreateCert(cf)
	case "ecc384":
		ec := crypto.NewECC()
		err = ec.CreateCert(cf)
		if err != nil {
			println("create ecc-cert error: " + err.Error())
			return
		}
	case "rsa2048":
		ec := crypto.NewRSA()
		err = ec.CreateCert(cf)
		if err != nil {
			println("create rsa-cert error: " + err.Error())
			return
		}
	case "rsa4096":
		ec := crypto.NewRSA()
		err = ec.CreateCert(cf)
		if err != nil {
			println("create rsa-cert error: " + err.Error())
			return
		}
	default:
		*cry = "ecc256"
		ec := crypto.NewECC()
		err = ec.CreateCert(cf)
		if err != nil {
			println("create ecc-cert error: " + err.Error())
			return
		}
	}
	println("create " + *cry + " cert file done.")
}
