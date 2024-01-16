package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func GenerateKey() {
	p, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		println(err.Error())
		return
	}
	derTxt, err := x509.MarshalECPrivateKey(p)
	if err != nil {
		println(err.Error())
		return
	}
	block := &pem.Block{
		Type:  "ecdsa private key",
		Bytes: derTxt,
	}
	b := pem.EncodeToMemory(block)
	os.WriteFile("eccprivate.pem", b, 0644)

	pub := p.PublicKey
	derTxt, err = x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		println("12123", err.Error())
		return
	}
	block = &pem.Block{
		Type:  "ecdsa public key",
		Bytes: derTxt,
	}
	b = pem.EncodeToMemory(block)
	os.WriteFile("eccpublic.pem", b, 0644)

}
func TestECC(t *testing.T) {
	bb := []byte("12319827391273hfsdkasdfafasdf")
	c := NewECC()
	c.GenerateKey(ECSecp384r1)
	c.ToFile("ecsecp384r1pub.pem", "ecsecp384r1pri.pem")
	c.GenerateKey(ECPrime256v1)
	c.ToFile("ecprime256v1pub.pem", "ecprime256v1pri.pem")
	// c.SetPublicKeyFromFile("ecpubkey.pem")
	// c.SetPrivateKeyFromFile("eccprime256v1.pem")
	x, err := c.Encode(bb)
	if err != nil {
		t.Fail()
		return
	}
	println(x.Base64String())

	z, err := c.Decode(x.Bytes())
	if err != nil {
		t.Fail()
		return
	}
	println(z)
	if z != string(bb) {
		t.Fail()
		return
	}
	b, err := c.Sign(bb)
	if err != nil {
		t.Fail()
		return
	}
	println(b.HexString())

	ok, err := c.VerySignFromHex(b.HexString(), bb)
	if err != nil {
		t.Fail()
		return
	}
	if !ok {
		t.Fail()
	}
}
