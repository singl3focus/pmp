//go:build !windows

package output

// encodeForClipboard is a no-op on non-Windows platforms where clipboard
// commands (pbcopy, xclip, xsel) accept UTF-8 natively.
func encodeForClipboard(data []byte) []byte {
	return data
}
