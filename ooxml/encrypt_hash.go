// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

func sha256Bytes(b []byte) []byte { s := sha256.Sum256(b); return s[:] }
func sha1Bytes(b []byte) []byte   { s := sha1.Sum(b); return s[:] }

// newHash returns the constructor matching a hashFunc, for HMAC.
func newHash(h hashFunc) func() hash.Hash {
	switch len(h(nil)) {
	case 64:
		return sha512.New
	case 48:
		return sha512.New384
	case 32:
		return sha256.New
	default:
		return sha1.New
	}
}
