package tunnel

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

// Encrypt encrypts data and returns the encrypted data
func Encrypt(data []byte, key string) ([]byte, error) {
	// generate a new AES cipher using our 32 byte key
	c, err := aes.NewCipher(hash(key))
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	// create a new byte array the size of the nonce
	// NOTE: we might use smaller nonce size in the future
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// NOTE: we prepend the nonce to the payload
	// we need to do this as we need the same nonce
	// to decrypt the payload when receiving it
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypt decrypts the payload and returns the decrypted data
func Decrypt(data []byte, key string) ([]byte, error) {
	// generate AES cipher for decrypting the message
	c, err := aes.NewCipher(hash(key))
	if err != nil {
		return nil, err
	}

	// we use GCM to encrypt the payload
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	// NOTE: we need to parse out nonce from the payload
	// we prepend the nonce to every encrypted payload
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// hash hahes the data into 32 bytes key and returns it
// hash uses sha256 underneath to hash the supplied key
func hash(key string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}
