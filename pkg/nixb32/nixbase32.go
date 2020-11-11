package nixb32

import (
	"strings"
)

const nixCharset string = "0123456789abcdfghijklmnpqrsvwxyz"

// Hash hashes a slice of bytes according to Nix's base32
func Hash(hexBytes []byte) (string, error) {
	hexLen := len(hexBytes)
	outLen := (len(hexBytes)*8-1)/5 + 1

	var hash strings.Builder
	hash.Grow(outLen)

	for n := outLen - 1; n >= 0; n-- {
		b := n * 5
		i := b / 8
		j := b % 8

		var v1 byte
		v1 = hexBytes[i] >> j

		var v2 byte
		if i >= hexLen-1 {
			v2 = 0
		} else {
			v2 = hexBytes[i+1] << (8 - j)
		}

		v := v1 | v2
		idx := int(v) % len(nixCharset)

		c := nixCharset[idx]

		hash.WriteByte(c)
	}

	return hash.String(), nil
}
