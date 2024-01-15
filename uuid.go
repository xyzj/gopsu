package gopsu

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"sync"
	"time"

	"github.com/xyzj/gopsu/crypto"
)

// A Time represents a time as the number of 100's of nanoseconds since 15 Oct
// 1582.
type Time int64

const (
	g1582ns100 = (2440587 - 2299160) * 86400 * 10000000 // 100s of a nanoseconds between epochs// epochs  Julian day of 1 Jan 1970 - Julian day of 15 Oct 1582
)

var (
	timeMu     sync.Mutex
	nodeMu     sync.Mutex
	lasttime   uint64          // last time we returned
	clockSeq   uint16          // clock sequence for this run
	timeNow    = time.Now      // for testing
	rander     = rand.Reader   // random function
	nodeID     [6]byte         // hardware for version 1 UUIDs
	zeroID     [6]byte         // nodeID with only 0's
	interfaces []net.Interface // cached list of interfaces
)

func encodeHex(dst, src []byte) {
	hex.Encode(dst, src[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], src[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], src[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], src[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], src[10:])
}

// GetTime returns the current Time (100s of nanoseconds since 15 Oct 1582) and
// clock sequence as well as adjusting the clock sequence as needed.  An error
// is returned if the current time cannot be determined.
func getTime() (Time, uint16, error) {
	defer timeMu.Unlock()
	timeMu.Lock()
	t := timeNow()

	// If we don't have a clock sequence already, set one.
	if clockSeq == 0 {
		setClockSequence(-1)
	}
	now := uint64(t.UnixNano()/100) + g1582ns100

	// If time has gone backwards with this clock sequence then we
	// increment the clock sequence
	if now <= lasttime {
		clockSeq = ((clockSeq + 1) & 0x3fff) | 0x8000
	}
	lasttime = now
	return Time(now), clockSeq, nil
}

func setClockSequence(seq int) {
	if seq == -1 {
		var b [2]byte
		randomBits(b[:]) // clock sequence
		seq = int(b[0])<<8 | int(b[1])
	}
	oldSeq := clockSeq
	clockSeq = uint16(seq&0x3fff) | 0x8000 // Set our variant
	if oldSeq != clockSeq {
		lasttime = 0
	}
}

func randomBits(b []byte) {
	if _, err := io.ReadFull(rander, b); err != nil {
		panic(err.Error()) // rand should never fail
	}
}

// getHardwareInterface returns the name and hardware address of interface name.
// If name is "" then the name and hardware address of one of the system's
// interfaces is returned.  If no interfaces are found (name does not exist or
// there are no interfaces) then "", nil is returned.
//
// Only addresses of at least 6 bytes are returned.
func getHardwareInterface(name string) (string, []byte) {
	if interfaces == nil {
		var err error
		interfaces, err = net.Interfaces()
		if err != nil {
			return "", nil
		}
	}
	for _, ifs := range interfaces {
		if len(ifs.HardwareAddr) >= 6 && (name == "" || name == ifs.Name) {
			return ifs.Name, ifs.HardwareAddr
		}
	}
	return "", nil
}

func setNodeInterface(name string) bool {
	iname, addr := getHardwareInterface(name) // null implementation for js
	if iname != "" && addr != nil {
		copy(nodeID[:], addr)
		return true
	}

	// We found no interfaces with a valid hardware address.  If name
	// does not specify a specific interface generate a random Node ID
	// (section 4.1.6)
	if name == "" {
		randomBits(nodeID[:])
		return true
	}
	return false
}

// GetUUID1 returns a Version 1 UUID based on the current NodeID and clock
// sequence, and the current time.  If the NodeID has not been set by SetNodeID
// or SetNodeInterface then it will be set automatically.  If the NodeID cannot
// be set NewUUID returns nil.  If clock sequence has not been set by
// SetClockSequence then it will be set automatically.  If GetTime fails to
// return the current NewUUID returns nil and an error.
//
// In most cases, New should be used.
func GetUUID1() string {
	var uuid [16]byte
	var buf [36]byte
	now, seq, err := getTime()
	if err != nil {
		encodeHex(buf[:], crypto.GetRandom(16))
		return String(buf[:])
	}

	timeLow := uint32(now & 0xffffffff)
	timeMid := uint16((now >> 32) & 0xffff)
	timeHi := uint16((now >> 48) & 0x0fff)
	timeHi |= 0x1000 // Version 1

	binary.BigEndian.PutUint32(uuid[0:], timeLow)
	binary.BigEndian.PutUint16(uuid[4:], timeMid)
	binary.BigEndian.PutUint16(uuid[6:], timeHi)
	binary.BigEndian.PutUint16(uuid[8:], seq)

	nodeMu.Lock()
	if nodeID == zeroID {
		setNodeInterface("")
	}
	copy(uuid[10:], nodeID[:])
	nodeMu.Unlock()

	encodeHex(buf[:], uuid[:])
	return String(buf[:])
}
