package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrNoKey = errors.New("config: no store key")

func ParseKey(name, b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, nil
	}
	k, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(k) != 32 {
		return nil, fmt.Errorf("%s must decode to 32 bytes, got %d", name, len(k))
	}
	return k, nil
}

func Seal(key []byte, plain string) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plain), nil), nil
}

func Open(key, sealed []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("config: sealed value too short")
	}
	plain, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// FingerprintWith identifies a value without revealing it, so two services (or a
// service and the store) can be shown to hold the same secret. It is keyed: a
// bare hash of a short password is a lookup away from the password itself.
func FingerprintWith(key []byte, value string) string {
	if value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, fingerprintKey(key))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))[:12]
}

func fingerprintKey(key []byte) []byte {
	sum := sha256.Sum256(append([]byte("kungal-config-fingerprint:"), key...))
	return sum[:]
}

func Hint(value string) string {
	r := []rune(value)
	if len(r) <= 8 {
		return "…"
	}
	return "…" + string(r[len(r)-4:])
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) == 0 {
		return nil, ErrNoKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
