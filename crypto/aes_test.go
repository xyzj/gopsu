package crypto

import (
	"crypto/aes"
	"testing"
)

var (
	s = "gopsu.GetRandomString(4096, true)"
)

func TestAES(t *testing.T) {
	clist := []struct {
		Name    string
		AesType AESType
	}{
		{"aes128cbc", AES128CBC},
		{"aes128cfb", AES128CFB},
		{"aes192cbc", AES192CBC},
		{"aes192cfb", AES192CFB},
		{"aes256cbc", AES256CBC},
		{"aes256cfb", AES256CFB},
	}
	// l := 10
	var key, iv = []byte("nNZT3xhtcKyykgBtsn7OAx0cymmNEPqE"), []byte("4qzB9DK6eFuSOMfB")
	println(string(key), string(iv))
	s = "PvNmdlAIIu"
	for _, x := range clist {
		t.Run(x.Name, func(t *testing.T) {
			c := NewAESWorker(x.AesType)
			c.SetKeyIV(key, iv)
			v, err := c.Encode([]byte(s))
			if err != nil {
				t.Fatalf("%s encode failed %s", x.Name, err.Error())
				return
			}
			println(v.Base64String())
			bb := v.Bytes()
			if x.AesType == AES128CFB || x.AesType == AES192CFB || x.AesType == AES256CFB {
				bb = bb[aes.BlockSize:]
			}
			ss, err := c.Decode(bb)
			if err != nil {
				t.Fatalf("%s encode failed %s", x.Name, err.Error())
				return
			}
			if s != ss {
				t.Fatalf("encode and decode is not match")
			}
		})
	}
}
