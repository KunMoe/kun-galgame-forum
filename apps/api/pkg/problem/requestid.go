package problem

import (
	"crypto/rand"
	"encoding/binary"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	HeaderRequestID   = "X-Request-ID"
	requestIDPrefix   = "req_"
	requestIDLocalKey = "request_id"
	crockford         = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

var requestIDPattern = regexp.MustCompile(`^req_[0-9A-HJKMNP-TV-Z]{26}$`)

func ValidRequestID(id string) bool {
	return requestIDPattern.MatchString(id)
}

func NewRequestID() string {
	return requestIDPrefix + newULID()
}

func RequestID(c fiber.Ctx) string {
	if v, ok := c.Locals(requestIDLocalKey).(string); ok && ValidRequestID(v) {
		return v
	}
	if v := c.Get(HeaderRequestID); ValidRequestID(v) {
		c.Locals(requestIDLocalKey, v)
		return v
	}
	id := NewRequestID()
	c.Locals(requestIDLocalKey, id)
	return id
}

func newULID() string {
	var raw [16]byte
	ms := uint64(time.Now().UnixMilli())
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	if _, err := rand.Read(raw[6:]); err != nil {
		binary.BigEndian.PutUint64(raw[6:14], uint64(time.Now().UnixNano()))
		binary.BigEndian.PutUint16(raw[14:], uint16(time.Now().UnixNano()))
	}
	return encodeULID(raw)
}

func encodeULID(id [16]byte) string {
	dst := [26]byte{
		crockford[(id[0]&224)>>5],
		crockford[id[0]&31],
		crockford[(id[1]&248)>>3],
		crockford[((id[1]&7)<<2)|((id[2]&192)>>6)],
		crockford[(id[2]&62)>>1],
		crockford[((id[2]&1)<<4)|((id[3]&240)>>4)],
		crockford[((id[3]&15)<<1)|((id[4]&128)>>7)],
		crockford[(id[4]&124)>>2],
		crockford[((id[4]&3)<<3)|((id[5]&224)>>5)],
		crockford[id[5]&31],
		crockford[(id[6]&248)>>3],
		crockford[((id[6]&7)<<2)|((id[7]&192)>>6)],
		crockford[(id[7]&62)>>1],
		crockford[((id[7]&1)<<4)|((id[8]&240)>>4)],
		crockford[((id[8]&15)<<1)|((id[9]&128)>>7)],
		crockford[(id[9]&124)>>2],
		crockford[((id[9]&3)<<3)|((id[10]&224)>>5)],
		crockford[id[10]&31],
		crockford[(id[11]&248)>>3],
		crockford[((id[11]&7)<<2)|((id[12]&192)>>6)],
		crockford[(id[12]&62)>>1],
		crockford[((id[12]&1)<<4)|((id[13]&240)>>4)],
		crockford[((id[13]&15)<<1)|((id[14]&128)>>7)],
		crockford[(id[14]&124)>>2],
		crockford[((id[14]&3)<<3)|((id[15]&224)>>5)],
		crockford[id[15]&31],
	}
	return string(dst[:])
}
