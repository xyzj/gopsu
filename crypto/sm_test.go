package crypto

import "testing"

func TestSM2(t *testing.T) {
	sss := "1267312shfskdfadfaf"
	c := NewSM2()
	c.GenerateKey()
	c.ToFile("sm2pub.pem", "sm2pri.pem")
	s, err := c.Encode([]byte(sss))
	if err != nil {
		t.Fail()
		return
	}
	s1, err := c.Decode(s.Bytes())
	if err != nil {
		t.Fail()
		return
	}
	if s1 != sss {
		t.Fail()
	}
}

func TestSM4(t *testing.T) {
	sss := "1267312shfskdfadfaf"
	var key, iv = []byte("nNZT3xhtcKyykgBtsn7OAx0cymmNEPqE"), []byte("4qzB9DK6eFuSOMfB")
	c := NewSM4(SM4CBC)
	c.SetKeyIV(key, iv)
	s, err := c.Encode([]byte(sss))
	if err != nil {
		t.Fail()
		return
	}
	s1, err := c.Decode(s.Bytes())
	if err != nil {
		t.Fail()
		return
	}
	if s1 != sss {
		t.Fail()
	}
}
