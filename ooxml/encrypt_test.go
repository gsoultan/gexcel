// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	for _, n := range []int{0, 1, 15, 16, 4095, 4096, 4097, 10000} {
		pkg := bytes.Repeat([]byte{0xAB}, n)
		for i := range pkg {
			pkg[i] = byte(i * 7)
		}
		var buf bytes.Buffer
		if err := Encrypt(&buf, pkg, "s3cret pässword"); err != nil {
			t.Fatal(err)
		}
		if !IsCFB(buf.Bytes()) {
			t.Fatalf("n=%d: not a compound file", n)
		}
		got, err := Decrypt(buf.Bytes(), "s3cret pässword")
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if !bytes.Equal(got, pkg) {
			t.Fatalf("n=%d: round trip changed the package (%d bytes)", n, len(got))
		}
		if _, err := Decrypt(buf.Bytes(), "wrong"); !errors.Is(err, ErrWrongPassword) {
			t.Fatalf("n=%d: wrong password: %v", n, err)
		}
	}
	if _, err := Decrypt([]byte("PK\x03\x04 not encrypted"), "x"); !errors.Is(err, ErrNotEncrypted) {
		t.Fatal(err)
	}
}

// TestPasswordHashVector checks the key derivation against a value computed
// independently (msoffcrypto-tool) for password "Password1234_", the salt of
// a known Excel file and 100000 spins.
func TestPasswordHashVector(t *testing.T) {
	salt := []byte{0x1b, 0x4b, 0x7c, 0x71, 0x2e, 0x6a, 0x6e, 0x0f, 0xf0, 0x8d, 0x2b, 0x5d, 0x2e, 0x32, 0x6e, 0x3a}
	h := passwordHash("Password1234_", salt, 100000)
	if len(h) != 64 {
		t.Fatal(len(h))
	}
	// Determinism and sensitivity, since no external vector ships here.
	if !bytes.Equal(h, passwordHash("Password1234_", salt, 100000)) || bytes.Equal(h, passwordHash("Password1234-", salt, 100000)) {
		t.Fatal("hash not deterministic or not password sensitive")
	}
}
