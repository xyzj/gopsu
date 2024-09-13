package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/tidwall/sjson"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

var (
	strcode  = "$argon2id$v=19$m=19456,t=2,p=1$HAK3hf4VULj3XiUH7tUwyg$ZfIQN8T4r8WdVCgIIWus8KmTMe+2ma22LQyWowECoGc"
	strplain = "affine"
)

func TruncFloat(f float64, n int) float64 {
	// 默认乘1
	d := float64(1)
	if n > 0 {
		// 10 的 n 次方
		d = math.Pow10(n)
	}
	// 截断 n 位小数:  math.trunc作用就是返回浮点数的整数部分
	f2 := math.Trunc(f*d) / d
	// 舍弃后面的0值: -1 参数表示保持原小数位数，千万要注意，如果你指定了位数就会四舍五入了
	fs := strconv.FormatFloat(f2, 'f', -1, 64)
	// 转换成float64
	f3, _ := strconv.ParseFloat(fs, 32)
	return f3
}

type aaa struct {
	AAA float64
}

func main() {
	a := float64(23.45623423445435345)
	b := fmt.Sprintf("%.3f", a)
	c, _ := strconv.ParseFloat(b, 64)
	s, _ := sjson.Set("", "aaa", c)
	ss := &aaa{
		AAA: math.Trunc(a*math.Pow10(1)+0.5) / math.Pow10(1),
	}
	println(reflect.TypeOf(1e5).String())
	bb, _ := json.Marshal(ss)
	println(a, b, c, s, string(bb), fmt.Sprintf("%.6f", 1e5))
}

func generateFromPassword(password string, p *params) (encodedHash string, err error) {
	salt, err := generateRandomBytes(p.saltLength)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Base64 encode the salt and hashed password.
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return a string using the standard encoded hash representation.
	encodedHash = fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.memory, p.iterations, p.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

func generateRandomBytes(n uint32) ([]byte, error) {
	// return []byte("1234567890123456"), nil
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func comparePasswordAndHash(password, encodedHash string) (match bool, err error) {
	// Extract the parameters, salt and derived key from the encoded password
	// hash.
	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	// Derive the key from the other password using the same parameters.
	otherHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Check that the contents of the hashed passwords are identical. Note
	// that we are using the subtle.ConstantTimeCompare() function for this
	// to help prevent timing attacks.
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (p *params, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	p = &params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}
