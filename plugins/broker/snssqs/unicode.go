package snssqs

import (
	"bytes"
	"unicode"

	"golang.org/x/text/unicode/rangetable"
)

var validSqsRunes = &unicode.RangeTable{}

func init() {
	validSqsRunes = genSqsRangeTable()
}

func genSqsRangeTable() *unicode.RangeTable {
	// #x9 | #xA | #xD | #x20 to #xD7FF | #xE000 to #xFFFD | #x10000 to #x10FFFF

	var buf bytes.Buffer

	buf.WriteRune(0x9)
	buf.WriteRune(0xa)
	buf.WriteRune(0xd)
	var r rune
	for r = 0x20; r <= 0xd7ff; r++ {
		buf.WriteRune(r)
	}
	for r = 0xe000; r <= 0xfffd; r++ {
		buf.WriteRune(r)
	}
	for r = 0x10000; r <= 0x10ffff; r++ {
		buf.WriteRune(r)
	}
	return rangetable.New([]rune(buf.String())...)
}
