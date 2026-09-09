package natsjskv

import (
	"encoding/base32"
	"strings"
)

// NatsKey is a convenience function to create a key for the nats kv store.
func (n *natsStore) NatsKey(table, microkey string) string {
	return n.NewKey(table, microkey, "").NatsKey()
}

// MicroKey is a convenience function to create a key for the micro interface.
func (n *natsStore) MicroKey(table, natskey string) string {
	return n.NewKey(table, "", natskey).MicroKey()
}

// MicroKeyFilter is a convenience function to create a key for the micro interface.
// It returns false if the key does not match the table, prefix or suffix.
func (n *natsStore) MicroKeyFilter(table, natskey string, prefix, suffix string) (string, bool) {
	k := n.NewKey(table, "", natskey)
	return k.MicroKey(), k.Check(table, prefix, suffix)
}

// Key represents a key in the store.
// They are used to convert nats keys (base32 encoded) to micro keys (plain text - no table prefix) and vice versa.
type Key struct {
	// Plain is the plain key as requested by the go-micro interface.
	Plain string
	// Full is the full key including the table prefix.
	Full string
	// Encoded is the base64 encoded key as used by the nats kv store.
	Encoded string
}

// NewKey creates a new key. Either plain or encoded must be set.
func (n *natsStore) NewKey(table string, plain, encoded string) *Key {
	k := &Key{
		Plain:   plain,
		Encoded: encoded,
	}

	switch {
	case k.Plain != "":
		k.Full = getKey(k.Plain, table)
		k.Encoded = encode(k.Full, n.encoding)
	case k.Encoded != "":
		k.Full = decode(k.Encoded, n.encoding)
		k.Plain = trimKey(k.Full, table)
	}

	return k
}

// NatsKey returns a key the nats kv store can work with.
func (k *Key) NatsKey() string {
	return k.Encoded
}

// MicroKey returns a key the micro interface can work with.
func (k *Key) MicroKey() string {
	return k.Plain
}

// Check returns false if the key does not match the table, prefix or suffix.
func (k *Key) Check(table, prefix, suffix string) bool {
	if table != "" && k.Full != getKey(k.Plain, table) {
		return false
	}

	if prefix != "" && !strings.HasPrefix(k.Plain, prefix) {
		return false
	}

	if suffix != "" && !strings.HasSuffix(k.Plain, suffix) {
		return false
	}

	return true
}

func encode(s string, alg string) string {
	switch alg {
	case "base32":
		return base32.StdEncoding.EncodeToString([]byte(s))
	default:
		return s
	}
}

func decode(s string, alg string) string {
	switch alg {
	case "base32":
		b, err := base32.StdEncoding.DecodeString(s)
		if err != nil {
			return s
		}

		return string(b)
	default:
		return s
	}
}

func getKey(key, table string) string {
	if table != "" {
		return table + "_" + key
	}

	return key
}

func trimKey(key, table string) string {
	if table != "" {
		return strings.TrimPrefix(key, table+"_")
	}

	return key
}
