// Package AESWrapper is a wrapper package
// for multiple crypto packages that will be used 
// together with the crypto/aes package to provide
// further functionality.
package AESWrapper

import (
	"crypto/rand"
	"crypto/aes"
	"crypto/cipher"
)

// GenerateKey is an alias for:
// GenerateKeyWithLength(32) since 256 bit 
// (32*8=256) is a common length for 
// an AES-Key.
func GenerateKey() []byte {
	return GenerateKeyWithLength(32)
}

// GenerateKeyWithLength returns a random AES-Key
// with the given length ([Bytes]). It uses 
// crypto/rand's rand.Read() function.
func GenerateKeyWithLength(length byte) []byte {
	key := make([]byte, length)
	rand.Read(key)

	return key
} 

// GenerateNonce is an alias for:
// GenerateNonceWithLength(12) since 12 Byte
// is the default nonce length using AES.
func GenerateNonce() []byte {
	return GenerateNonceWithLength(12)
}

// GenerateNonceWithLength returns a random AES-Nonce
// with the given length ([Bytes]). It uses
// crypto/rand's rand.Read() function.
func GenerateNonceWithLength(length byte) []byte {
	nonce := make([]byte, length)
	rand.Read(nonce)

	return nonce
}

// GenerateAESGCM generates a cipher.AEAD with
// the given key. 
// It returns the generated cipher.AEAD + an error
// (if any occured)
func GenerateAESGCM(AESKey []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(AESKey)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	return aesgcm, err
} 

// Encrypt encrypts the given plaintext using the
// AES cipher with the given key and nonce. 
// It returns the cipher text + an error (if any occured)
func Encrypt(plaintext []byte, key []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{0}, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{0}, err
	}

	return aesgcm.Seal(nil, nonce, plaintext, nil), nil
}

// Decrypt decrypts the given ciphertext using the
// AES cipher with the given key and nonce.
// It returns the plain text + an error (if one occured)
func Decrypt(ciphertext []byte, key []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{0}, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte{0}, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return []byte{0}, err
	}

	return plaintext, nil
}