// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"bytes"
	"testing"
)

func TestCFBRoundTrip(t *testing.T) {
	small := bytes.Repeat([]byte("ab"), 100)   // mini stream
	large := bytes.Repeat([]byte("xyz"), 3000) // 9000 bytes, regular sectors
	exact := bytes.Repeat([]byte{7}, 4096)     // at the cutoff: regular sectors
	var buf bytes.Buffer
	if err := WriteCFB(&buf, []CFBStream{{"EncryptedPackage", large}, {"EncryptionInfo", small}, {"Exact", exact}}); err != nil {
		t.Fatal(err)
	}
	if buf.Len()%512 != 0 || !IsCFB(buf.Bytes()) {
		t.Fatalf("bad container: %d bytes", buf.Len())
	}
	got, err := ReadCFB(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]byte{"EncryptedPackage": large, "EncryptionInfo": small, "Exact": exact} {
		if !bytes.Equal(got[name], want) {
			t.Errorf("%s: got %d bytes, want %d", name, len(got[name]), len(want))
		}
	}
}

func TestCFBLess(t *testing.T) {
	if !cfbLess("Exact", "EncryptionInfo") || cfbLess("EncryptionInfo", "Exact") {
		t.Error("shorter names must sort first")
	}
	if !cfbLess("abc", "ABD") || cfbLess("ABD", "abc") {
		t.Error("compare upper-cased")
	}
}
