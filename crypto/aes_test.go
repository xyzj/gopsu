package crypto

import (
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
		{Name: "aes128cbc", AesType: AES128CBC},
		{Name: "aes128cfb", AesType: AES128CFB},
		{Name: "aes192cbc", AesType: AES192CBC},
		{Name: "aes192cfb", AesType: AES192CFB},
		{Name: "aes256cbc", AesType: AES256CBC},
		{Name: "aes256cfb", AesType: AES256CFB},
		{Name: "AES128ECB", AesType: AES128ECB},
		{Name: "AES192ECB", AesType: AES192ECB},
		{Name: "AES256ECB", AesType: AES256ECB},
	}
	// l := 10
	var key, iv = "nNZT3xhtcKyykgBtsn7OAx0cymmNEPqE", "4qzB9DK6eFuSOMfB"
	println(string(key), string(iv))
	s = "PvNmsssssdlAIIu"
	for _, x := range clist {
		t.Run(x.Name, func(t *testing.T) {
			c := NewAES(x.AesType)
			c.SetPadding(Pkcs7Padding)
			c.SetKeyIV(key, iv)
			v, err := c.Encode([]byte(s))
			if err != nil {
				t.Fatalf("%s encode failed %s", x.Name, err.Error())
				return
			}
			if v.Base64String() != c.Encrypt(s) {
				t.Fail()
				return
			}
			bb := v.Bytes()
			ss, err := c.Decode(bb)
			if err != nil {
				t.Fatalf("%s decode failed %s", x.Name, err.Error())
				return
			}
			// println(ss)
			// ss, err = c.Decode(bb)
			// if err != nil {
			// 	t.Fatalf("%s decode failed %s", x.Name, err.Error())
			// 	return
			// }
			if s != ss {
				t.Fatalf("encode and decode is not match")
			}
		})
	}
}
