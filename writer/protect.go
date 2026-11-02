// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package writer

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// hashPassword derives the ISO/IEC 29500 sheet/workbook protection hash
// (SHA-512, 100000 spins, 16-byte random salt). It returns the algorithm
// name, hash and salt as base64, and the spin count.
func hashPassword(password string) (alg, hash, salt string, spin int) {
	var saltBytes [16]byte
	if _, err := rand.Read(saltBytes[:]); err != nil {
		panic(err)
	}
	spin = 100000
	h := HashPasswordWithSalt(password, saltBytes[:], spin)
	return "SHA-512", base64.StdEncoding.EncodeToString(h), base64.StdEncoding.EncodeToString(saltBytes[:]), spin
}

// HashPasswordWithSalt computes the protection hash for a password with a
// given salt and spin count: H0 = SHA512(salt + UTF-16LE(password)),
// Hn = SHA512(Hn-1 + LE32(n-1)).
func HashPasswordWithSalt(password string, salt []byte, spin int) []byte {
	u := utf16.Encode([]rune(password))
	buf := make([]byte, 0, len(salt)+2*len(u))
	buf = append(buf, salt...)
	for _, c := range u {
		buf = binary.LittleEndian.AppendUint16(buf, c)
	}
	h := sha512.Sum512(buf)
	cur := h[:]
	var iter [4]byte
	for i := 0; i < spin; i++ {
		binary.LittleEndian.PutUint32(iter[:], uint32(i))
		next := sha512.Sum512(append(append([]byte{}, cur...), iter[:]...))
		cur = next[:]
	}
	return cur
}

// VerifyPassword checks a password against a stored SHA-512 hash.
func VerifyPassword(password, hashB64, saltB64 string, spin int) bool {
	hash, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}
	got := HashPasswordWithSalt(password, salt, spin)
	if len(got) != len(hash) {
		return false
	}
	for i := range got {
		if got[i] != hash[i] {
			return false
		}
	}
	return true
}

// LegacyPasswordHash computes the 16-bit hash Excel 97-2003 used for sheet
// and workbook protection (the "password" attribute), as 4 upper-case hex
// digits.
func LegacyPasswordHash(password string) string {
	var hash uint16
	runes := []rune(password)
	for i := len(runes) - 1; i >= 0; i-- {
		hash = (hash>>14)&1 | (hash<<1)&0x7fff
		hash ^= uint16(runes[i])
	}
	hash = (hash>>14)&1 | (hash<<1)&0x7fff
	hash ^= uint16(len(runes))
	hash ^= 0xCE4B
	return fmt.Sprintf("%04X", hash)
}
