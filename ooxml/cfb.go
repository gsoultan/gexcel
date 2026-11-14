// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package ooxml

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"unicode/utf16"
)

// A minimal Compound File Binary (OLE2) container, enough for the two
// streams of an encrypted Office package (EncryptionInfo and
// EncryptedPackage): 512-byte sectors, one directory, a mini stream for
// small streams, no red-black balancing beyond a sorted sibling chain.

const (
	cfbSectorShift     = 9
	cfbSectorSize      = 1 << cfbSectorShift
	cfbMiniSectorShift = 6
	cfbMiniSectorSize  = 1 << cfbMiniSectorShift
	cfbMiniCutoff      = 4096

	cfbEndOfChain = 0xFFFFFFFE
	cfbFreeSect   = 0xFFFFFFFF
	cfbFATSect    = 0xFFFFFFFD
	cfbNoStream   = 0xFFFFFFFF

	cfbObjStorage = 1
	cfbObjStream  = 2
	cfbObjRoot    = 5
)

var cfbSignature = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// IsCFB reports whether data starts with the compound file signature.
func IsCFB(data []byte) bool {
	return len(data) >= 8 && string(data[:8]) == string(cfbSignature)
}

// CFBStream is one named stream of a compound file.
type CFBStream struct {
	Name string
	Data []byte
}

type cfbEntry struct {
	name   string
	typ    byte
	left   uint32
	right  uint32
	child  uint32
	start  uint32
	size   uint64
	stream []byte
}

// WriteCFB writes streams (all children of the root storage) as a compound
// file. Names must be unique; the root storage holds the mini stream.
func WriteCFB(w io.Writer, streams []CFBStream) error {
	sort.Slice(streams, func(i, j int) bool { return cfbLess(streams[i].Name, streams[j].Name) })
	// Lay out the streams: small ones in the mini stream, big ones in sectors.
	var mini []byte
	var big [][]byte // one per big stream, each padded to a sector multiple
	entries := []*cfbEntry{{name: "Root Entry", typ: cfbObjRoot}}
	for _, s := range streams {
		if len(s.Name) == 0 || len(s.Name) > 31 {
			return fmt.Errorf("cfb: invalid stream name %q", s.Name)
		}
		e := &cfbEntry{name: s.Name, typ: cfbObjStream, size: uint64(len(s.Data)), stream: s.Data}
		entries = append(entries, e)
	}
	// Mini stream contents and mini FAT.
	var miniFAT []uint32
	for _, e := range entries[1:] {
		if e.size >= cfbMiniCutoff || e.size == 0 {
			continue
		}
		e.start = uint32(len(miniFAT))
		n := (len(e.stream) + cfbMiniSectorSize - 1) / cfbMiniSectorSize
		for i := range n {
			if i == n-1 {
				miniFAT = append(miniFAT, cfbEndOfChain)
			} else {
				miniFAT = append(miniFAT, e.start+uint32(i)+1)
			}
		}
		mini = append(mini, e.stream...)
		mini = append(mini, make([]byte, n*cfbMiniSectorSize-len(e.stream))...)
	}
	for _, e := range entries[1:] {
		if e.size == 0 {
			e.start = cfbEndOfChain
		}
	}
	// Sector plan: [directory][miniFAT][mini stream][big streams...][FAT]
	sectorsOf := func(n int) int { return (n + cfbSectorSize - 1) / cfbSectorSize }
	dirSectors := sectorsOf(len(entries) * 128)
	miniFATSectors := sectorsOf(len(miniFAT) * 4)
	miniSectors := sectorsOf(len(mini))
	next := uint32(0)
	dirStart := next
	next += uint32(dirSectors)
	miniFATStart := uint32(cfbEndOfChain)
	if miniFATSectors > 0 {
		miniFATStart = next
		next += uint32(miniFATSectors)
	}
	root := entries[0]
	root.start = cfbEndOfChain
	if miniSectors > 0 {
		root.start = next
		root.size = uint64(len(mini))
		next += uint32(miniSectors)
	}
	for _, e := range entries[1:] {
		if e.size >= cfbMiniCutoff {
			e.start = next
			n := sectorsOf(len(e.stream))
			padded := make([]byte, n*cfbSectorSize)
			copy(padded, e.stream)
			big = append(big, padded)
			next += uint32(n)
		}
	}
	// FAT sectors: each holds 128 entries and describes itself too.
	dataSectors := int(next)
	fatSectors := 0
	for sectorsOf((dataSectors+fatSectors)*4) > fatSectors {
		fatSectors++
	}
	if fatSectors > 109 {
		return errors.New("cfb: file too large for this writer")
	}
	total := dataSectors + fatSectors
	fat := make([]uint32, fatSectors*128)
	for i := range fat {
		fat[i] = cfbFreeSect
	}
	chain := func(start uint32, n int) {
		for i := range n {
			if i == n-1 {
				fat[int(start)+i] = cfbEndOfChain
			} else {
				fat[int(start)+i] = start + uint32(i) + 1
			}
		}
	}
	chain(dirStart, dirSectors)
	if miniFATSectors > 0 {
		chain(miniFATStart, miniFATSectors)
	}
	if miniSectors > 0 {
		chain(root.start, miniSectors)
	}
	for _, e := range entries[1:] {
		if e.size >= cfbMiniCutoff {
			chain(e.start, sectorsOf(len(e.stream)))
		}
	}
	for i := range fatSectors {
		fat[dataSectors+i] = cfbFATSect
	}
	// Directory tree: the root's child is the middle entry of the sorted
	// streams; siblings form a chain (valid, if unbalanced, for a few entries).
	for i, e := range entries {
		e.left, e.right, e.child = cfbNoStream, cfbNoStream, cfbNoStream
		if i == 0 && len(entries) > 1 {
			e.child = 1
		}
		if i >= 1 && i+1 < len(entries) {
			e.right = uint32(i + 1)
		}
	}
	// Header.
	hdr := make([]byte, cfbSectorSize)
	copy(hdr, cfbSignature)
	binary.LittleEndian.PutUint16(hdr[24:], 0x003E) // minor version
	binary.LittleEndian.PutUint16(hdr[26:], 0x0003) // major version 3
	binary.LittleEndian.PutUint16(hdr[28:], 0xFFFE) // byte order
	binary.LittleEndian.PutUint16(hdr[30:], cfbSectorShift)
	binary.LittleEndian.PutUint16(hdr[32:], cfbMiniSectorShift)
	binary.LittleEndian.PutUint32(hdr[44:], uint32(fatSectors))
	binary.LittleEndian.PutUint32(hdr[48:], dirStart)
	binary.LittleEndian.PutUint32(hdr[56:], cfbMiniCutoff)
	binary.LittleEndian.PutUint32(hdr[60:], miniFATStart)
	binary.LittleEndian.PutUint32(hdr[64:], uint32(miniFATSectors))
	binary.LittleEndian.PutUint32(hdr[68:], cfbEndOfChain) // DIFAT start
	binary.LittleEndian.PutUint32(hdr[72:], 0)             // DIFAT sectors
	for i := range 109 {
		v := uint32(cfbFreeSect)
		if i < fatSectors {
			v = uint32(dataSectors + i)
		}
		binary.LittleEndian.PutUint32(hdr[76+i*4:], v)
	}
	out := make([]byte, 0, (total+1)*cfbSectorSize)
	out = append(out, hdr...)
	// Directory.
	dir := make([]byte, dirSectors*cfbSectorSize)
	for i := range dir[len(entries)*128:] {
		dir[len(entries)*128+i] = 0
	}
	for i, e := range entries {
		b := dir[i*128 : (i+1)*128]
		u := utf16.Encode([]rune(e.name))
		for k, c := range u {
			binary.LittleEndian.PutUint16(b[k*2:], c)
		}
		binary.LittleEndian.PutUint16(b[64:], uint16(len(u)*2+2))
		b[66] = e.typ
		b[67] = 1 // black
		binary.LittleEndian.PutUint32(b[68:], e.left)
		binary.LittleEndian.PutUint32(b[72:], e.right)
		binary.LittleEndian.PutUint32(b[76:], e.child)
		binary.LittleEndian.PutUint32(b[116:], e.start)
		binary.LittleEndian.PutUint64(b[120:], e.size)
	}
	// Unused directory slots are "free" entries: type 0, no siblings.
	for i := len(entries); i < len(dir)/128; i++ {
		b := dir[i*128 : (i+1)*128]
		binary.LittleEndian.PutUint32(b[68:], cfbNoStream)
		binary.LittleEndian.PutUint32(b[72:], cfbNoStream)
		binary.LittleEndian.PutUint32(b[76:], cfbNoStream)
	}
	out = append(out, dir...)
	if miniFATSectors > 0 {
		mf := make([]byte, miniFATSectors*cfbSectorSize)
		for i := range mf[len(miniFAT)*4:] {
			mf[len(miniFAT)*4+i] = 0xFF
		}
		for i, v := range miniFAT {
			binary.LittleEndian.PutUint32(mf[i*4:], v)
		}
		out = append(out, mf...)
	}
	if miniSectors > 0 {
		ms := make([]byte, miniSectors*cfbSectorSize)
		copy(ms, mini)
		out = append(out, ms...)
	}
	for _, b := range big {
		out = append(out, b...)
	}
	fb := make([]byte, fatSectors*cfbSectorSize)
	for i, v := range fat {
		binary.LittleEndian.PutUint32(fb[i*4:], v)
	}
	out = append(out, fb...)
	_, err := w.Write(out)
	return err
}

// cfbLess orders directory names as the format requires: shorter first,
// then by upper-cased UTF-16 code units.
func cfbLess(a, b string) bool {
	ua, ub := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	if len(ua) != len(ub) {
		return len(ua) < len(ub)
	}
	for i := range ua {
		ca, cb := upper16(ua[i]), upper16(ub[i])
		if ca != cb {
			return ca < cb
		}
	}
	return false
}

func upper16(c uint16) uint16 {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// ReadCFB parses a compound file and returns its streams by name (storages
// are flattened: every stream anywhere in the tree is returned by its own
// name, which is enough for Office encryption containers).
func ReadCFB(data []byte) (map[string][]byte, error) {
	if !IsCFB(data) || len(data) < cfbSectorSize {
		return nil, errors.New("cfb: not a compound file")
	}
	shift := binary.LittleEndian.Uint16(data[30:])
	if shift != 9 && shift != 12 {
		return nil, errors.New("cfb: unsupported sector size")
	}
	ss := 1 << shift
	miniShift := binary.LittleEndian.Uint16(data[32:])
	if miniShift != 6 {
		return nil, errors.New("cfb: unsupported mini sector size")
	}
	numFAT := binary.LittleEndian.Uint32(data[44:])
	dirStart := binary.LittleEndian.Uint32(data[48:])
	cutoff := binary.LittleEndian.Uint32(data[56:])
	miniFATStart := binary.LittleEndian.Uint32(data[60:])
	numMiniFAT := binary.LittleEndian.Uint32(data[64:])
	difatStart := binary.LittleEndian.Uint32(data[68:])
	numDIFAT := binary.LittleEndian.Uint32(data[72:])
	sector := func(n uint32) ([]byte, error) {
		off := (int(n) + 1) * ss
		if n >= cfbFATSect || off+ss > len(data) {
			return nil, fmt.Errorf("cfb: sector %d out of range", n)
		}
		return data[off : off+ss], nil
	}
	// FAT sector list from the header DIFAT and any DIFAT sectors.
	var fatSectors []uint32
	for i := range 109 {
		v := binary.LittleEndian.Uint32(data[76+i*4:])
		if v == cfbFreeSect {
			break
		}
		fatSectors = append(fatSectors, v)
	}
	for n, i := difatStart, uint32(0); i < numDIFAT && n != cfbEndOfChain && n != cfbFreeSect; i++ {
		s, err := sector(n)
		if err != nil {
			return nil, err
		}
		for k := 0; k < ss/4-1; k++ {
			v := binary.LittleEndian.Uint32(s[k*4:])
			if v == cfbFreeSect {
				continue
			}
			fatSectors = append(fatSectors, v)
		}
		n = binary.LittleEndian.Uint32(s[ss-4:])
	}
	if uint32(len(fatSectors)) < numFAT {
		return nil, errors.New("cfb: truncated FAT")
	}
	fat := make([]uint32, 0, len(fatSectors)*(ss/4))
	for _, fs := range fatSectors[:numFAT] {
		s, err := sector(fs)
		if err != nil {
			return nil, err
		}
		for k := 0; k < ss/4; k++ {
			fat = append(fat, binary.LittleEndian.Uint32(s[k*4:]))
		}
	}
	readChain := func(start uint32, size int) ([]byte, error) {
		var out []byte
		seen := 0
		for n := start; n != cfbEndOfChain && n != cfbFreeSect; {
			s, err := sector(n)
			if err != nil {
				return nil, err
			}
			out = append(out, s...)
			if seen++; seen > len(fat) {
				return nil, errors.New("cfb: FAT chain loop")
			}
			if int(n) >= len(fat) {
				return nil, errors.New("cfb: FAT chain out of range")
			}
			n = fat[n]
		}
		if size >= 0 && size <= len(out) {
			out = out[:size]
		}
		return out, nil
	}
	dir, err := readChain(dirStart, -1)
	if err != nil {
		return nil, err
	}
	var miniFAT []uint32
	if numMiniFAT > 0 {
		mf, err := readChain(miniFATStart, int(numMiniFAT)*ss)
		if err != nil {
			return nil, err
		}
		for k := 0; k+4 <= len(mf); k += 4 {
			miniFAT = append(miniFAT, binary.LittleEndian.Uint32(mf[k:]))
		}
	}
	var miniStream []byte
	streams := map[string][]byte{}
	for i := 0; (i+1)*128 <= len(dir); i++ {
		b := dir[i*128 : (i+1)*128]
		typ := b[66]
		nameLen := int(binary.LittleEndian.Uint16(b[64:]))
		if typ == 0 || nameLen < 2 || nameLen > 64 {
			continue
		}
		u := make([]uint16, (nameLen-2)/2)
		for k := range u {
			u[k] = binary.LittleEndian.Uint16(b[k*2:])
		}
		name := string(utf16.Decode(u))
		start := binary.LittleEndian.Uint32(b[116:])
		size := binary.LittleEndian.Uint64(b[120:])
		if typ == cfbObjRoot {
			miniStream, err = readChain(start, int(size))
			if err != nil {
				return nil, err
			}
			continue
		}
		if typ != cfbObjStream {
			continue
		}
		if size >= uint64(cutoff) {
			d, err := readChain(start, int(size))
			if err != nil {
				return nil, err
			}
			streams[name] = d
			continue
		}
		var d []byte
		seen := 0
		for n := start; n != cfbEndOfChain && n != cfbFreeSect && uint64(len(d)) < size; {
			off := int(n) * cfbMiniSectorSize
			if off+cfbMiniSectorSize > len(miniStream) || int(n) >= len(miniFAT) {
				return nil, errors.New("cfb: mini stream out of range")
			}
			d = append(d, miniStream[off:off+cfbMiniSectorSize]...)
			if seen++; seen > len(miniFAT) {
				return nil, errors.New("cfb: mini FAT chain loop")
			}
			n = miniFAT[n]
		}
		if uint64(len(d)) > size {
			d = d[:size]
		}
		streams[name] = d
	}
	return streams, nil
}
