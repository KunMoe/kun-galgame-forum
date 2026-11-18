package config

import (
	"bytes"
	"testing"
)

var testKey = bytes.Repeat([]byte{7}, 32)

func TestSealOpenRoundTrip(t *testing.T) {
	sealed, err := Seal(testKey, "hunter2")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(testKey, sealed)
	if err != nil || got != "hunter2" {
		t.Fatalf("Open = %q, %v", got, err)
	}
	if _, err := Open(bytes.Repeat([]byte{8}, 32), sealed); err == nil {
		t.Fatal("a different key must not open the value")
	}
	if _, err := Seal(nil, "x"); err != ErrNoKey {
		t.Fatalf("Seal without key = %v, want ErrNoKey", err)
	}
}

func TestFingerprintIsKeyedAndStable(t *testing.T) {
	a, b := FingerprintWith(testKey, "v"), FingerprintWith(testKey, "v")
	if a == "" || a != b {
		t.Fatalf("fingerprint not stable: %q %q", a, b)
	}
	if FingerprintWith(bytes.Repeat([]byte{8}, 32), "v") == a {
		t.Fatal("fingerprint must depend on the key")
	}
	if FingerprintWith(testKey, "") != "" {
		t.Fatal("an empty value has no fingerprint")
	}
}

func TestParseKey(t *testing.T) {
	if k, err := ParseKey("K", ""); k != nil || err != nil {
		t.Fatalf("empty = %v, %v", k, err)
	}
	if _, err := ParseKey("K", "c2hvcnQ="); err == nil {
		t.Fatal("a short key must be rejected")
	}
}
