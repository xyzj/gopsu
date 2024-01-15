package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"sync"

	"github.com/tjfoc/gmsm/sm3"
)

// HashWorker hash算法
type HashWorker struct {
	locker   sync.Mutex
	hash     hash.Hash
	workType HashType
}

// SetHMACKey 设置hmac算法的key
func (w *HashWorker) SetHMACKey(key []byte) {
	switch w.workType {
	case HashHMACSHA1:
		w.hash = hmac.New(sha1.New, key)
	case HashHMACSHA256:
		w.hash = hmac.New(sha256.New, key)
	}
}

// Hash 计算哈希值
func (w *HashWorker) Hash(b []byte) CValue {
	w.locker.Lock()
	defer w.locker.Unlock()
	w.hash.Reset()
	w.hash.Write(b)
	return CValue(w.hash.Sum(nil))
}

// NewHashWorker 创建一个新的hash算法器
func NewHashWorker(t HashType) *HashWorker {
	w := &HashWorker{
		locker:   sync.Mutex{},
		workType: t,
	}
	switch t {
	case HashMD5:
		w.hash = md5.New()
	case HashHMACSHA1:
		w.hash = hmac.New(sha1.New, []byte{})
	case HashHMACSHA256:
		w.hash = hmac.New(sha256.New, []byte{})
	case HashSHA1:
		w.hash = sha1.New()
	case HashSHA256:
		w.hash = sha256.New()
	case HashSHA512:
		w.hash = sha512.New()
	case HashSM3:
		w.hash = sm3.New()
	}
	return w
}
