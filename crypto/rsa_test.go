package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func RSAGenKey(bits int) error {
	/*
		生成私钥
	*/
	//1、使用RSA中的GenerateKey方法生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	//2、通过X509标准将得到的RAS私钥序列化为：ASN.1 的DER编码字符串
	privateStream := x509.MarshalPKCS1PrivateKey(privateKey)
	//3、将私钥字符串设置到pem格式块中
	block1 := pem.Block{
		Type:  "private key",
		Bytes: privateStream,
	}
	//4、通过pem将设置的数据进行编码，并写入磁盘文件
	fPrivate, err := os.Create("privateKey.pem")
	if err != nil {
		return err
	}
	defer fPrivate.Close()
	err = pem.Encode(fPrivate, &block1)
	if err != nil {
		return err
	}

	/*
		生成公钥
	*/
	publicKey := privateKey.PublicKey
	publicStream, _ := x509.MarshalPKIXPublicKey(&publicKey)
	//publicStream:=x509.MarshalPKCS1PublicKey(&publicKey)
	block2 := pem.Block{
		Type:  "public key",
		Bytes: publicStream,
	}
	fPublic, err := os.Create("publicKey.pem")
	if err != nil {
		return err
	}
	defer fPublic.Close()
	pem.Encode(fPublic, &block2)
	return nil
}
func TestRSA(t *testing.T) {
	// RSAGenKey(4096)
	sss := "1267312shfskdfadfaf"
	c := NewRSA()
	c.GenerateKey(RSA2048)
	c.ToFile("rsa2048pub.pem", "rsa2048pri.pem")
	c.GenerateKey(RSA4096)
	c.ToFile("rsa4096pub.pem", "rsa4096pri.pem")
	// err := c.SetPublicKeyFromFile("publicKey.pem")
	// if err != nil {
	// 	t.Fatal("set public key error " + err.Error())
	// 	return
	// }
	// err = c.SetPrivateKeyFromFile("privateKey.pem")
	// if err != nil {
	// 	t.Fatal("set private key error " + err.Error())
	// 	return
	// }
	if v, err := c.Encode([]byte(sss)); err != nil {
		t.Fatal("encode error " + err.Error())
	} else {
		xs, err := c.Decode(v.Bytes())
		if err != nil {
			t.Fatal("decode error " + err.Error())
			return
		}
		if xs != sss {
			t.Fatal("encode decode not match")
		}
	}
}

func TestSign(t *testing.T) {
	sss := "1267312shfskdfadfaf" //gopsu.GetRandomString(30002, true) // "1267312shfskdfadfaf"
	c := NewRSA()
	c.SetPublicKeyFromFile("publicKey.pem")
	c.SetPrivateKeyFromFile("privateKey.pem")
	x, err := c.Sign([]byte(sss))
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	println(x.HexString())
	z, err := c.VerySign(x.Bytes(), []byte(sss))
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	if !z {
		t.Fail()
		return
	}
	z, err = c.VerySignFromBase64(x.Base64String(), []byte(sss))
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	if !z {
		t.Fail()
	}
}
