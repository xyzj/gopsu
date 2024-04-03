package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
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
	bb := []byte(`{"token":"604213a4-9e4e-11ee-8e42-0242ac110004","ts":1711704865}`)
	c := NewECC()
	// c.GenerateKey(ECSecp384r1)
	// c.ToFile("ecsecp384r1pub.pem", "ecsecp384r1pri.pem")
	// c.GenerateKey(ECPrime256v1)
	// c.ToFile("ecprime256v1pub.pem", "ecprime256v1pri.pem")
	err := c.SetPublicKey("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEMgp3YvXro++Q3LwgRc0AHI9lSS7G+4EeU66CKW1YfCn/5MMClMX7memrWxR5HtOQRqCKRE5LDqgy6F0poSqj7w==")
	if err != nil {
		t.Fatal(err)
		return
	}
	err = c.SetPrivateKey("MHcCAQEEIFVEEYqRAWy7rg5GO5nyTFxsAP9WmV1nSrlOsum5/GDAoAoGCCqGSM49AwEHoUQDQgAEMgp3YvXro++Q3LwgRc0AHI9lSS7G+4EeU66CKW1YfCn/5MMClMX7memrWxR5HtOQRqCKRE5LDqgy6F0poSqj7w==")
	if err != nil {
		t.Fatal(err)
		return
	}
	x, err := c.Encode(bb)
	if err != nil {
		t.Fatal(err)
		return
	}
	println("encode: ", x.Base64String(), x.Len())
	// z, err := c.Decode(x.Bytes())
	bbb, _ := base64.StdEncoding.DecodeString("BAOaY58rralImATiSP6baDPah76PdoImsgBGAY08sezEuO18kJRFreeFjaqYznyRK2mCXuEUuz5Y2fZ+cghCSzsB/8byjXnxz15gFxyGx5muJECX+ELJY3NjC9x63NYYoG/H1dPKs2ILTkqKpunPTCQRG1bVVf58NZZwMU9X9uzZK81qK1B5uKmzk2ARdfkLucHYcJ8=")
	println(len(bbb))
	z, err := c.DecodeBase64("BAOaY58rralImATiSP6baDPah76PdoImsgBGAY08sezEuO18kJRFreeFjaqYznyRK2mCXuEUuz5Y2fZ+cghCSzsB/8byjXnxz15gFxyGx5muJECX+ELJY3NjC9x63NYYoG/H1dPKs2ILTkqKpunPTCQRG1bVVf58NZZwMU9X9uzZK81qK1B5uKmzk2ARdfkLucHYcJ8=")
	if err != nil {
		t.Fatal(err)
		return
	}
	println("decode: ", z)
	if z != string(bb) {
		t.Fatal(err)
		return
	}
	// b, err := c.Sign(bb)
	// if err != nil {
	// 	t.Fail()
	// 	return
	// }
	// println(b.HexString())

	// ok, err := c.VerifySignFromHex(b.HexString(), bb)
	// if err != nil {
	// 	t.Fail()
	// 	return
	// }
	// if !ok {
	// 	t.Fail()
	// }
}

func TestEncode(t *testing.T) {
	// s := "BObrPMyP5QeXufnmnwfUXE3DX7V39kzi7SUndfC0YF4a8i9dL6Vtuzs9HzkjGaPHyRMMTLnF3F+1/aO72CNzJK7HGKuGH4q/nZ3895ItxYktcNr3NUIpBiqDF4FP1D9fTQvfREZKZYF1xJSu5WgSTr7upqcF8WH2liLnv3urGo3JRh0A0imivWqreHGnBvORRV2bjTwZTvoQybq6tycF5UYNdPRZJRZ2hKwqxOIakrYx"
	c := NewECC()
	err := c.SetPrivateKeyFromFile("private.pem")
	if err != nil {
		t.Fatal(err)
		return
	}
	err = c.SetPublicKeyFromFile("public.pem")
	if err != nil {
		t.Fatal(err)
		return
	}
	b, _ := os.ReadFile("s.json")
	// c.SetPrivateKey("MHcCAQEEIFVEEYqRAWy7rg5GO5nyTFxsAP9WmV1nSrlOsum5/GDAoAoGCCqGSM49AwEHoUQDQgAEMgp3YvXro++Q3LwgRc0AHI9lSS7G+4EeU66CKW1YfCn/5MMClMX7memrWxR5HtOQRqCKRE5LDqgy6F0poSqj7w==")
	s1, err := c.Encode(b)
	if err != nil {
		t.Fatal(err)
		return
	}
	println("encode1: ", s1.Base64String())
	os.WriteFile("file2.enc", s1.Bytes(), 0664)
}

func TestDecode(t *testing.T) {
	b, err := os.ReadFile("file2.enc")
	if err != nil {
		t.Fatal(err)
		return
	}

	// s := "BObrPMyP5QeXufnmnwfUXE3DX7V39kzi7SUndfC0YF4a8i9dL6Vtuzs9HzkjGaPHyRMMTLnF3F+1/aO72CNzJK7HGKuGH4q/nZ3895ItxYktcNr3NUIpBiqDF4FP1D9fTQvfREZKZYF1xJSu5WgSTr7upqcF8WH2liLnv3urGo3JRh0A0imivWqreHGnBvORRV2bjTwZTvoQybq6tycF5UYNdPRZJRZ2hKwqxOIakrYx"
	c := NewECC()
	err = c.SetPrivateKeyFromFile("private.pem")
	if err != nil {
		t.Fatal(err)
		return
	}
	err = c.SetPublicKeyFromFile("public.pem")
	if err != nil {
		t.Fatal(err)
		return
	}
	// c.SetPrivateKey("MHcCAQEEIFVEEYqRAWy7rg5GO5nyTFxsAP9WmV1nSrlOsum5/GDAoAoGCCqGSM49AwEHoUQDQgAEMgp3YvXro++Q3LwgRc0AHI9lSS7G+4EeU66CKW1YfCn/5MMClMX7memrWxR5HtOQRqCKRE5LDqgy6F0poSqj7w==")
	s1, err := c.Decode(b)
	if err != nil {
		t.Fatal(err)
		return
	}
	println("decode2: ", s1, "===")
	os.WriteFile("s2.json", []byte(s1), 0664)
}

func TestECCSign(t *testing.T) {
	s := `{"token": "604213a4-9e4e-11ee-8e42-0242ac110004", "ts": 1711704865}`
	c := NewECC()
	// c.SetPrivateKeyFromFile("ecprime256v1pri.pem")
	c.SetPublicKeyFromFile("public.pem")
	// x, _ := c.Sign([]byte(s))
	// println(x.Base64String())
	s1 := "MEUCIQCnTqQA0l17qUFCn90S+T5rh9q/jPkS/IsaGNpAv1a3rwIgX5a5rz1Jy7NoWGdH339Slmd4BMwayLFDdWH1YMxYhbU="
	b, _ := c.VerifySignFromBase64(s1, []byte(s))
	println(b)
}

func TestECCert(t *testing.T) {
	c := NewECC()
	// c.SetPrivateKeyFromFile("cert-key.ec.pem")
	err := c.CreateCert(nil, nil)
	if err != nil {
		t.Fatal(err)
		return
	}
}
