package helper

import "strings"

// BytesIter helps process byte strings
type BytesIter struct {
	data string
	pos  int
}

func NewBytesIter(data string) *BytesIter {
	return &BytesIter{
		data: strings.TrimPrefix(data, "0x"),
		pos:  0,
	}
}

func (b *BytesIter) NextUint160() string {
	return b.NextBytes(20)
}

func (b *BytesIter) NextUint256() string {
	return b.NextBytes(32)
}

func (b *BytesIter) NextBytes(n int) string {
	if b.pos+n*2 > len(b.data) {
		return ""
	}
	result := b.data[b.pos : b.pos+n*2]
	b.pos += n * 2
	return "0x" + result
}

func (b *BytesIter) Rest() string {
	if b.pos >= len(b.data) {
		return ZX
	}
	return "0x" + b.data[b.pos:]
}
