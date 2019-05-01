// Package RSAWrapper is a wrapper class
// for the encryption/rsa package. It provides
// little more functionality and will be used
// mostly due to snugness.
package RSAWrapper

import (
	"crypto/rsa"
	"crypto/rand"
)

// GenerateKey is an alias for:
// GenerateKeyWithLength(2048) which
// is a common length for RSA-Keys.
func GenerateKey() (*rsa.PrivateKey, error) {
	return GenerateKeyWithLength(2048)
}

// GenerateKeyWithLength is the wrapper function 
// for rsa.GenerateKey and returns a 
// rsa.PrivateKey-Pointer with the given length.
func GenerateKeyWithLength(length int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, length)
}