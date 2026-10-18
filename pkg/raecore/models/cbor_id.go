package models

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

// GenerateUUIDv7 generates a time-ordered RFC 9562 UUIDv7 string in pure Go.
func GenerateUUIDv7() string {
	var buf [16]byte
	ms := uint64(time.Now().UTC().UnixMilli())
	buf[0] = byte(ms >> 40)
	buf[1] = byte(ms >> 32)
	buf[2] = byte(ms >> 24)
	buf[3] = byte(ms >> 16)
	buf[4] = byte(ms >> 8)
	buf[5] = byte(ms)

	_, _ = rand.Read(buf[6:])
	// Version 7 (0111)
	buf[6] = (buf[6] & 0x0f) | 0x70
	// Variant RFC 4122 (10xx)
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// GenerateULID generates a lexicographically sortable 26-char ULID string in pure Go.
func GenerateULID() string {
	ms := uint64(time.Now().UTC().UnixMilli())
	var rnd [10]byte
	_, _ = rand.Read(rnd[:])

	var out [26]byte
	// 48-bit timestamp -> 10 chars
	out[0] = crockfordAlphabet[(ms>>45)&0x1F]
	out[1] = crockfordAlphabet[(ms>>40)&0x1F]
	out[2] = crockfordAlphabet[(ms>>35)&0x1F]
	out[3] = crockfordAlphabet[(ms>>30)&0x1F]
	out[4] = crockfordAlphabet[(ms>>25)&0x1F]
	out[5] = crockfordAlphabet[(ms>>20)&0x1F]
	out[6] = crockfordAlphabet[(ms>>15)&0x1F]
	out[7] = crockfordAlphabet[(ms>>10)&0x1F]
	out[8] = crockfordAlphabet[(ms>>5)&0x1F]
	out[9] = crockfordAlphabet[ms&0x1F]

	// 80-bit randomness -> 16 chars
	for i := 0; i < 16; i++ {
		byteIdx := (i * 5) / 8
		bitOffset := (i * 5) % 8
		val := uint16(rnd[byteIdx]) << 8
		if byteIdx+1 < 10 {
			val |= uint16(rnd[byteIdx+1])
		}
		idx := (val >> (11 - bitOffset)) & 0x1F
		out[10+i] = crockfordAlphabet[idx]
	}
	return string(out[:])
}

// EncodeDeterministicCBOR encodes a string-keyed map into RFC 8949 deterministic CBOR bytes
// (keys sorted by canonical UTF-8 byte order).
func EncodeDeterministicCBOR(record map[string]string) []byte {
	var buf bytes.Buffer
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	// RFC 8949 Section 4.2.1: sort by length first, then lexicographically by bytes
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) < len(keys[j])
		}
		return keys[i] < keys[j]
	})

	writeCBORHeader(&buf, 5, uint64(len(keys))) // Major type 5: map
	for _, k := range keys {
		writeCBORText(&buf, k)
		writeCBORText(&buf, record[k])
	}
	return buf.Bytes()
}

func writeCBORHeader(buf *bytes.Buffer, majorType byte, val uint64) {
	mt := majorType << 5
	switch {
	case val < 24:
		buf.WriteByte(mt | byte(val))
	case val <= 0xFF:
		buf.WriteByte(mt | 24)
		buf.WriteByte(byte(val))
	case val <= 0xFFFF:
		buf.WriteByte(mt | 25)
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(val))
		buf.Write(b[:])
	default:
		buf.WriteByte(mt | 26)
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(val))
		buf.Write(b[:])
	}
}

func writeCBORText(buf *bytes.Buffer, s string) {
	writeCBORHeader(buf, 3, uint64(len(s))) // Major type 3: text string
	buf.WriteString(s)
}

// ComputeVersionCID computes a Multihash CID over RFC 8949 deterministic CBOR encoding.
func ComputeVersionCID(record map[string]string) string {
	cborBytes := EncodeDeterministicCBOR(record)
	sum := sha256.Sum256(cborBytes)
	// Multihash prefix 0x12 (sha2-256) 0x20 (32 bytes)
	return "mh:1220" + hex.EncodeToString(sum[:])
}
