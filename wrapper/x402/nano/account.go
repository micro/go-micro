package nano

import (
	"errors"
	"strings"
)

const accountAlphabet = "13456789abcdefghijkmnopqrstuwxyz"

// ErrInvalidAccount is returned for an account that is malformed or whose
// checksum does not match its public key.
var ErrInvalidAccount = errors.New("nano: invalid nano account")

// EncodeAccount encodes a 32-byte public key as a nano_ account with its
// BLAKE2b-40 checksum.
func EncodeAccount(pub []byte) (string, error) {
	if len(pub) != 32 {
		return "", ErrInvalidAccount
	}
	// 4 zero pad bits + 256 key bits = 260 bits = 52 characters.
	var b strings.Builder
	b.WriteString("nano_")
	b.WriteString(encode5(append([]byte{0, 0, 0}, pub...), 52))
	sum := blake2bSum(pub, 5)
	rev := []byte{sum[4], sum[3], sum[2], sum[1], sum[0]}
	b.WriteString(encode5(rev, 8))
	return b.String(), nil
}

// DecodeAccount checks the prefix, alphabet and checksum of a nano_ (or
// legacy xrb_) account and returns its public key.
func DecodeAccount(account string) ([]byte, error) {
	var body string
	switch {
	case strings.HasPrefix(account, "nano_"):
		body = account[5:]
	case strings.HasPrefix(account, "xrb_"):
		body = account[4:]
	default:
		return nil, ErrInvalidAccount
	}
	if len(body) != 60 || (body[0] != '1' && body[0] != '3') {
		return nil, ErrInvalidAccount
	}
	keyBits, ok := decode5(body[:52], 35)
	if !ok {
		return nil, ErrInvalidAccount
	}
	// The first 3 bytes hold the 4 pad bits plus nothing else.
	if keyBits[0] != 0 || keyBits[1] != 0 || keyBits[2] != 0 {
		return nil, ErrInvalidAccount
	}
	pub := keyBits[3:]
	chk, ok := decode5(body[52:], 5)
	if !ok {
		return nil, ErrInvalidAccount
	}
	sum := blake2bSum(pub, 5)
	for i := 0; i < 5; i++ {
		if chk[i] != sum[4-i] {
			return nil, ErrInvalidAccount
		}
	}
	return append([]byte(nil), pub...), nil
}

// ValidateAccount reports whether account is a well-formed nano_ account
// with a correct checksum.
func ValidateAccount(account string) error {
	_, err := DecodeAccount(account)
	return err
}

// encode5 writes the low chars*5 bits of the big-endian number in b.
func encode5(b []byte, chars int) string {
	out := make([]byte, chars)
	for i := chars - 1; i >= 0; i-- {
		bitPos := (chars - 1 - i) * 5
		var v byte
		for j := 0; j < 5; j++ {
			p := bitPos + j
			byteIdx := len(b) - 1 - p/8
			if byteIdx >= 0 && b[byteIdx]&(1<<(uint(p)%8)) != 0 {
				v |= 1 << uint(j)
			}
		}
		out[i] = accountAlphabet[v]
	}
	return string(out)
}

// decode5 reads base32 characters into a big-endian number of size bytes.
// Bits that do not fit in size bytes must be zero.
func decode5(s string, size int) ([]byte, bool) {
	out := make([]byte, size)
	for i := 0; i < len(s); i++ {
		v := strings.IndexByte(accountAlphabet, s[i])
		if v < 0 {
			return nil, false
		}
		bitPos := (len(s) - 1 - i) * 5
		for j := 0; j < 5; j++ {
			if v&(1<<uint(j)) == 0 {
				continue
			}
			p := bitPos + j
			byteIdx := size - 1 - p/8
			if byteIdx < 0 {
				return nil, false
			}
			out[byteIdx] |= 1 << (uint(p) % 8)
		}
	}
	return out, true
}
