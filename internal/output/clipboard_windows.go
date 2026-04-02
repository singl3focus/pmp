//go:build windows

package output

import (
	"encoding/binary"
	"unicode/utf16"
)

// encodeForClipboard converts UTF-8 bytes to UTF-16LE for Windows clip.exe,
// which expects UTF-16LE input to handle non-ASCII characters correctly.
func encodeForClipboard(data []byte) []byte {
	runes := []rune(string(data))
	pairs := utf16.Encode(runes)
	buf := make([]byte, 2+len(pairs)*2) // BOM + encoded pairs
	// Write UTF-16LE BOM so clip.exe recognises the encoding.
	binary.LittleEndian.PutUint16(buf[0:2], 0xFEFF)
	for i, u := range pairs {
		binary.LittleEndian.PutUint16(buf[2+i*2:], u)
	}
	return buf
}
