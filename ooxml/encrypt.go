// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
)

// ECMA-376 agile encryption (MS-OFFCRYPTO 2.3.4.10): AES-256-CBC over the
// package in 4096-byte segments, keys derived from the password with
// SHA-512 and 100000 spins, HMAC-SHA512 data integrity. This is what Excel
// 2013 and later write for "Encrypt with Password", and what Excel 2010,
// LibreOffice and every current reader open.

const (
	encSpinCount   = 100000
	encSegmentSize = 4096
	encKeyBytes    = 32 // AES-256
	encBlockSize   = 16
	encHashSize    = 64 // SHA-512
	encSaltSize    = 16
)

var (
	blockKeyVerifierInput = []byte{0xfe, 0xa7, 0xd2, 0x76, 0x3b, 0x4b, 0x9e, 0x79}
	blockKeyVerifierValue = []byte{0xd7, 0xaa, 0x0f, 0x6d, 0x30, 0x61, 0x34, 0x4e}
	blockKeyEncryptedKey  = []byte{0x14, 0x6e, 0x0b, 0xe7, 0xab, 0xac, 0xd0, 0xd6}
	blockKeyHmacKey       = []byte{0x5f, 0xb2, 0xad, 0x01, 0x0c, 0xb9, 0xe1, 0xf6}
	blockKeyHmacValue     = []byte{0xa0, 0x67, 0x7f, 0x02, 0xb2, 0x2c, 0x84, 0x33}
)

// Errors.
var (
	ErrWrongPassword         = errors.New("ooxml: wrong password")
	ErrNotEncrypted          = errors.New("ooxml: not an encrypted package")
	ErrUnsupportedEncryption = errors.New("ooxml: unsupported encryption (only agile AES/SHA-512 and SHA-1/256/384 variants are read)")
)

// Encrypt wraps a package (the bytes of an .xlsx zip) in an encrypted
// compound file that opens with password.
func Encrypt(w io.Writer, pkg []byte, password string) error {
	if password == "" {
		return errors.New("ooxml: empty password")
	}
	if len(pkg) > MaxCFBStreamSize-encSegmentSize {
		return fmt.Errorf("ooxml: package of %d bytes is too large to encrypt", len(pkg))
	}
	var keySalt, pwSalt, verifierInput [encSaltSize]byte
	secret := make([]byte, encKeyBytes)
	hmacKey := make([]byte, encHashSize)
	for _, b := range [][]byte{keySalt[:], pwSalt[:], verifierInput[:], secret, hmacKey} {
		if _, err := rand.Read(b); err != nil {
			return err
		}
	}
	// Package stream: 8-byte plaintext length, then the encrypted segments.
	enc := make([]byte, 8, 8+len(pkg)+encBlockSize)
	binary.LittleEndian.PutUint64(enc, uint64(len(pkg)))
	for i := 0; i*encSegmentSize < len(pkg); i++ {
		seg := pkg[i*encSegmentSize:]
		if len(seg) > encSegmentSize {
			seg = seg[:encSegmentSize]
		}
		iv := segmentIV(keySalt[:], uint32(i))
		enc = append(enc, aesCBC(secret, iv, pad16(seg), true)...)
	}
	mac := hmac.New(sha512.New, hmacKey)
	mac.Write(enc)
	hmacValue := mac.Sum(nil)
	// Password key encryptor.
	h := passwordHash(password, pwSalt[:], encSpinCount)
	verifierHash := sha512.Sum512(verifierInput[:])
	info := encryptionInfo{
		KeyData: keyData{SaltSize: encSaltSize, BlockSize: encBlockSize, KeyBits: encKeyBytes * 8, HashSize: encHashSize,
			CipherAlgorithm: "AES", CipherChaining: "ChainingModeCBC", HashAlgorithm: "SHA512", SaltValue: b64(keySalt[:])},
		DataIntegrity: dataIntegrity{
			EncryptedHmacKey:   b64(aesCBC(secret, blockIV(keySalt[:], blockKeyHmacKey), pad16(hmacKey), true)),
			EncryptedHmacValue: b64(aesCBC(secret, blockIV(keySalt[:], blockKeyHmacValue), pad16(hmacValue), true)),
		},
	}
	info.KeyEncryptors.Encryptor = keyEncryptor{URI: "http://schemas.microsoft.com/office/2006/keyEncryptor/password",
		Key: encryptedKey{SpinCount: encSpinCount, SaltSize: encSaltSize, BlockSize: encBlockSize, KeyBits: encKeyBytes * 8, HashSize: encHashSize,
			CipherAlgorithm: "AES", CipherChaining: "ChainingModeCBC", HashAlgorithm: "SHA512", SaltValue: b64(pwSalt[:]),
			EncryptedVerifierHashInput: b64(aesCBC(deriveKey(h, blockKeyVerifierInput), pwSalt[:], verifierInput[:], true)),
			EncryptedVerifierHashValue: b64(aesCBC(deriveKey(h, blockKeyVerifierValue), pwSalt[:], verifierHash[:], true)),
			EncryptedKeyValue:          b64(aesCBC(deriveKey(h, blockKeyEncryptedKey), pwSalt[:], secret, true)),
		}}
	xmlBytes, err := xml.Marshal(info)
	if err != nil {
		return err
	}
	infoStream := make([]byte, 8)
	binary.LittleEndian.PutUint16(infoStream[0:], 4) // version major
	binary.LittleEndian.PutUint16(infoStream[2:], 4) // version minor: agile
	binary.LittleEndian.PutUint32(infoStream[4:], 0x40)
	infoStream = append(infoStream, xmlBytes...)
	return WriteCFB(w, []CFBStream{{"EncryptionInfo", infoStream}, {"EncryptedPackage", enc}})
}

// Decrypt opens an encrypted compound file with password and returns the
// package bytes. It returns ErrWrongPassword when the verifier does not
// match and ErrNotEncrypted when data is not a compound file.
func Decrypt(data []byte, password string) ([]byte, error) {
	if !IsCFB(data) {
		return nil, ErrNotEncrypted
	}
	streams, err := ReadCFB(data)
	if err != nil {
		return nil, err
	}
	infoStream, enc := streams["EncryptionInfo"], streams["EncryptedPackage"]
	if len(infoStream) < 8 || len(enc) < 8 {
		return nil, ErrNotEncrypted
	}
	major, minor := binary.LittleEndian.Uint16(infoStream), binary.LittleEndian.Uint16(infoStream[2:])
	if major != 4 || minor != 4 {
		return nil, fmt.Errorf("%w: EncryptionInfo version %d.%d", ErrUnsupportedEncryption, major, minor)
	}
	var info encryptionInfo
	if err := xml.Unmarshal(infoStream[8:], &info); err != nil {
		return nil, fmt.Errorf("ooxml: EncryptionInfo: %w", err)
	}
	kd, ke := info.KeyData, info.KeyEncryptors.Encryptor.Key
	hashFn, ok := hashByName(ke.HashAlgorithm)
	if !ok || kd.CipherAlgorithm != "AES" || kd.CipherChaining != "ChainingModeCBC" || ke.CipherAlgorithm != "AES" || ke.CipherChaining != "ChainingModeCBC" {
		return nil, ErrUnsupportedEncryption
	}
	keyHash, ok := hashByName(kd.HashAlgorithm)
	if !ok {
		return nil, ErrUnsupportedEncryption
	}
	pwSalt, err := unb64(ke.SaltValue)
	if err != nil {
		return nil, err
	}
	keySalt, err := unb64(kd.SaltValue)
	if err != nil {
		return nil, err
	}
	h := passwordHashWith(hashFn, password, pwSalt, ke.SpinCount)
	keyLen := ke.KeyBits / 8
	iv := fitBlock(pwSalt, ke.BlockSize)
	verifierInput, err := decryptField(ke.EncryptedVerifierHashInput, deriveKeyWith(hashFn, h, blockKeyVerifierInput, keyLen), iv)
	if err != nil {
		return nil, err
	}
	verifierValue, err := decryptField(ke.EncryptedVerifierHashValue, deriveKeyWith(hashFn, h, blockKeyVerifierValue, keyLen), iv)
	if err != nil {
		return nil, err
	}
	want := hashFn(verifierInput[:min(len(verifierInput), ke.SaltSize)])
	if len(verifierValue) < len(want) || subtle.ConstantTimeCompare(verifierValue[:len(want)], want) != 1 {
		return nil, ErrWrongPassword
	}
	secret, err := decryptField(ke.EncryptedKeyValue, deriveKeyWith(hashFn, h, blockKeyEncryptedKey, keyLen), iv)
	if err != nil {
		return nil, err
	}
	secret = secret[:min(len(secret), kd.KeyBits/8)]
	// Integrity.
	if info.DataIntegrity.EncryptedHmacKey != "" {
		hk, err := decryptField(info.DataIntegrity.EncryptedHmacKey, secret, blockIVWith(keyHash, keySalt, blockKeyHmacKey, kd.BlockSize))
		if err != nil {
			return nil, err
		}
		hv, err := decryptField(info.DataIntegrity.EncryptedHmacValue, secret, blockIVWith(keyHash, keySalt, blockKeyHmacValue, kd.BlockSize))
		if err != nil {
			return nil, err
		}
		mac := hmac.New(newHash(keyHash), hk[:min(len(hk), kd.HashSize)])
		mac.Write(enc)
		if sum := mac.Sum(nil); subtle.ConstantTimeCompare(hv[:min(len(hv), len(sum))], sum) != 1 {
			return nil, errors.New("ooxml: encrypted package failed its integrity check")
		}
	}
	size := binary.LittleEndian.Uint64(enc)
	body := enc[8:]
	if size > uint64(len(body)) {
		return nil, errors.New("ooxml: encrypted package is truncated")
	}
	out := make([]byte, 0, size)
	for i := 0; i*encSegmentSize < len(body); i++ {
		seg := body[i*encSegmentSize:]
		if len(seg) > encSegmentSize {
			seg = seg[:encSegmentSize]
		}
		if len(seg)%encBlockSize != 0 {
			return nil, errors.New("ooxml: encrypted segment is not block aligned")
		}
		out = append(out, aesCBC(secret, blockIVWith(keyHash, keySalt, le32(uint32(i)), kd.BlockSize), seg, false)...)
	}
	return out[:size], nil
}

func decryptField(b64v string, key, iv []byte) ([]byte, error) {
	c, err := unb64(b64v)
	if err != nil {
		return nil, err
	}
	if len(c) == 0 || len(c)%encBlockSize != 0 {
		return nil, errors.New("ooxml: bad encrypted field")
	}
	return aesCBC(key, iv, c, false), nil
}

type hashFunc func([]byte) []byte

func hashByName(name string) (hashFunc, bool) {
	switch name {
	case "SHA512", "SHA-512":
		return func(b []byte) []byte { s := sha512.Sum512(b); return s[:] }, true
	case "SHA384", "SHA-384":
		return func(b []byte) []byte { s := sha512.Sum384(b); return s[:] }, true
	case "SHA256", "SHA-256":
		return sha256Bytes, true
	case "SHA1", "SHA-1":
		return sha1Bytes, true
	}
	return nil, false
}

// passwordHash runs the agile key derivation: H0 = H(salt + UTF-16LE(pw)),
// Hn = H(LE32(n-1) + Hn-1).
func passwordHash(password string, salt []byte, spin int) []byte {
	return passwordHashWith(func(b []byte) []byte { s := sha512.Sum512(b); return s[:] }, password, salt, spin)
}

func passwordHashWith(h hashFunc, password string, salt []byte, spin int) []byte {
	u := utf16.Encode([]rune(password))
	buf := make([]byte, 0, len(salt)+2*len(u))
	buf = append(buf, salt...)
	for _, c := range u {
		buf = binary.LittleEndian.AppendUint16(buf, c)
	}
	cur := h(buf)
	iter := make([]byte, 4+len(cur))
	for i := 0; i < spin; i++ {
		binary.LittleEndian.PutUint32(iter, uint32(i))
		copy(iter[4:], cur)
		cur = h(iter)
	}
	return cur
}

// deriveKey turns the password hash into the key of one encrypted field:
// H(hash + blockKey), cut or zero-padded to the key size.
func deriveKey(h, blockKey []byte) []byte {
	return deriveKeyWith(func(b []byte) []byte { s := sha512.Sum512(b); return s[:] }, h, blockKey, encKeyBytes)
}

func deriveKeyWith(hf hashFunc, h, blockKey []byte, keyLen int) []byte {
	return fitBlock(hf(append(append([]byte{}, h...), blockKey...)), keyLen)
}

// blockIV is the IV of a keyData-encrypted field: H(keySalt + blockKey) cut
// to the block size.
func blockIV(keySalt, blockKey []byte) []byte {
	return blockIVWith(func(b []byte) []byte { s := sha512.Sum512(b); return s[:] }, keySalt, blockKey, encBlockSize)
}

func blockIVWith(hf hashFunc, keySalt, blockKey []byte, blockSize int) []byte {
	return fitBlock(hf(append(append([]byte{}, keySalt...), blockKey...)), blockSize)
}

func segmentIV(keySalt []byte, i uint32) []byte { return blockIV(keySalt, le32(i)) }

func le32(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

// fitBlock cuts b to n bytes or pads it with 0x36 as the specification says.
func fitBlock(b []byte, n int) []byte {
	if len(b) >= n {
		return b[:n]
	}
	out := make([]byte, n)
	copy(out, b)
	for i := len(b); i < n; i++ {
		out[i] = 0x36
	}
	return out
}

// pad16 zero-pads b to a whole number of AES blocks. Callers pass at most one
// 4096-byte segment or a hash; anything larger is a bug.
func pad16(b []byte) []byte {
	if len(b)%encBlockSize == 0 {
		return b
	}
	if len(b) > encSegmentSize {
		panic("ooxml: pad16 called with more than one segment")
	}
	out := make([]byte, (len(b)/encBlockSize+1)*encBlockSize)
	copy(out, b)
	return out
}

func aesCBC(key, iv, data []byte, encrypt bool) []byte {
	blk, err := aes.NewCipher(key)
	if err != nil {
		panic(err) // key sizes are fixed by the caller
	}
	out := make([]byte, len(data))
	if encrypt {
		cipher.NewCBCEncrypter(blk, iv).CryptBlocks(out, data)
	} else {
		cipher.NewCBCDecrypter(blk, iv).CryptBlocks(out, data)
	}
	return out
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func unb64(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("ooxml: EncryptionInfo: bad base64: %w", err)
	}
	return b, nil
}

// EncryptionInfo XML (namespaces as Office writes them).
type encryptionInfo struct {
	XMLName       xml.Name      `xml:"http://schemas.microsoft.com/office/2006/encryption encryption"`
	KeyData       keyData       `xml:"keyData"`
	DataIntegrity dataIntegrity `xml:"dataIntegrity"`
	KeyEncryptors keyEncryptors `xml:"keyEncryptors"`
}

type keyData struct {
	SaltSize        int    `xml:"saltSize,attr"`
	BlockSize       int    `xml:"blockSize,attr"`
	KeyBits         int    `xml:"keyBits,attr"`
	HashSize        int    `xml:"hashSize,attr"`
	CipherAlgorithm string `xml:"cipherAlgorithm,attr"`
	CipherChaining  string `xml:"cipherChaining,attr"`
	HashAlgorithm   string `xml:"hashAlgorithm,attr"`
	SaltValue       string `xml:"saltValue,attr"`
}

type dataIntegrity struct {
	EncryptedHmacKey   string `xml:"encryptedHmacKey,attr"`
	EncryptedHmacValue string `xml:"encryptedHmacValue,attr"`
}

type keyEncryptors struct {
	Encryptor keyEncryptor `xml:"keyEncryptor"`
}

type keyEncryptor struct {
	URI string       `xml:"uri,attr"`
	Key encryptedKey `xml:"http://schemas.microsoft.com/office/2006/keyEncryptor/password encryptedKey"`
}

type encryptedKey struct {
	SpinCount                  int    `xml:"spinCount,attr"`
	SaltSize                   int    `xml:"saltSize,attr"`
	BlockSize                  int    `xml:"blockSize,attr"`
	KeyBits                    int    `xml:"keyBits,attr"`
	HashSize                   int    `xml:"hashSize,attr"`
	CipherAlgorithm            string `xml:"cipherAlgorithm,attr"`
	CipherChaining             string `xml:"cipherChaining,attr"`
	HashAlgorithm              string `xml:"hashAlgorithm,attr"`
	SaltValue                  string `xml:"saltValue,attr"`
	EncryptedVerifierHashInput string `xml:"encryptedVerifierHashInput,attr"`
	EncryptedVerifierHashValue string `xml:"encryptedVerifierHashValue,attr"`
	EncryptedKeyValue          string `xml:"encryptedKeyValue,attr"`
}

var _ = bytes.MinRead
