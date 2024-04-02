package crypto

import (
	"encoding/base64"
	"testing"
)

func TestSM2(t *testing.T) {
	sss := "1267312shfskdfadfaf"
	c := NewSM2()
	// c.GenerateKey()
	err := c.SetPublicKeyFromFile("sm2pub.pem")
	if err != nil {
		t.Fatal(err)
		return
	}
	err = c.SetPrivateKeyFromFile("sm2pri.pem")
	if err != nil {
		t.Fatal(err)
		return
	}
	c.ToFile("sm2pub.pem", "sm2pri.pem")
	// c.SetPrivateKey("AKFHqhHU7xXlgzc5U/c4PsxxUa62DvjkAyBAxcq3uhDY")
	// c.SetPublicKey("BPWIR6XwO8j41zr153bbudpdwLQlIPrcE/PhiKeSx/ENrMJP4lYB7cGTEDBv0ZNdGvYdZrvN2VXBNTzkK91ASVQ=")
	s, err := c.Encode([]byte(sss))
	if err != nil {
		t.Fatal(err)
		return
	}
	println(s.HexString())
	b, _ := base64.StdEncoding.DecodeString("BFNvwaQ5aSeV2Kwzakts46HiuB8x/8Yk7UiFG8ap/8mghHoQJWOkWjXEFZL6Vu450wM0qCSxjJ0XNn8u3ERSlv4Clw2dbVO+t99WP7TEJ0np977AOEEQv3Js/q+pPDC/q0OsDkaBhnOjAfgxGzoDfSfwSlyUACGw9Jdn8g26Uoz469IRH8wX+034B+r5fQf7dv3UoRdwkWEdFQq26NTSRvhxLiNubVJxxz6CxxrGkxmvP10wBy7tLBO7LgXKfHX0IOB+aYDLv4FhhV+/9kV5K0QYLykpKbdaTQBpedEvpF7br6CVnSpaX8fKn6msJUU9jxRG2fzZXlQ1QhcHCfd96S5Qa98ZGY16009kiVafe714eMCedpb9XLo8ckaUvY42Cda8Q6ZIFSdU8KID0LACKwXxOVFThWyHZT2MJ7Z7pSMXoFquDEdPxFug2NMRJH1lyQKdIJxOMD9aai9cDg==")
	s1, err := c.Decode(b)
	// s1, err := c.Decode(s.Bytes())
	if err != nil {
		t.Fatal(err)
		return
	}
	if s1 != sss {
		t.Fail()
	}
	println(s1)
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
