package socks

import (
	"crypto"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
)

// ReceiveRSAPublicKey receives a PublicKey using the gob.Decoder
// dec and writes it to the rsa.PublicKey publ.
// If an error occurs it will be returned, otherwise the
// return value will be nil.
func ReceiveRSAPublicKey(dec *gob.Decoder, publ *rsa.PublicKey) error {
	return dec.Decode(&publ)
}

// ReceiveRSAMessage is an alias for:
// ReceiveRSAMessageWithLabel(dec, priv, publ, []byte(""))
func ReceiveRSAMessage(dec *gob.Decoder, priv *rsa.PrivateKey, publ *rsa.PublicKey) ([]byte, error) {
	return ReceiveRSAMessageWithLabel(dec, priv, publ, []byte(""))
}

// ReceiveRSAMessageWithLabel receives a struct of type RSAMessage
// using the gob.Decoder dec, decrypts it using the rsa.PrivateKey
// priv and checks its signature with the given rsa.PublicKey publ.
// It returns the message's plaintext + an error (if one occured).
func ReceiveRSAMessageWithLabel(dec *gob.Decoder, priv *rsa.PrivateKey, publ *rsa.PublicKey, lbl []byte) ([]byte, error) {
	var msg RSAMessage
	var plaintext []byte
	var err error
	var sha_hash [32]byte

	if err := dec.Decode(&msg); err != nil {
		return []byte{0}, err
	}

	plaintext, err = rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, msg.Ciphertext, lbl)
	if err != nil {
		return []byte{0}, err
	}

	sha_hash = sha256.Sum256(plaintext)
	err = rsa.VerifyPKCS1v15(publ, crypto.SHA256, sha_hash[:], msg.Signature)

	return plaintext, err
}

// ReceiveAESMessage is an alias for:
// ReceiveAESMessageWithLabel(dec, aesgcm, priv, []byte(""))
func ReceiveAESMessage(dec *gob.Decoder, aesgcm cipher.AEAD, priv *rsa.PrivateKey) ([]byte, error) {
	return ReceiveAESMessageWithLabel(dec, aesgcm, priv, []byte(""))
}

// ReceiveAESMessageWithLabel receives a struct of type AESMessage
// using the gob.Decoder dec, decrypts it's nonce using the given
// rsa.PrivateKey priv and finally decrypts the message using the
// cipher.AEAD aesgcm.
// It returns the message's plaintext + an error (if any occured)
func ReceiveAESMessageWithLabel(dec *gob.Decoder, aesgcm cipher.AEAD, priv *rsa.PrivateKey, lbl []byte) ([]byte, error) {
	var msg AESMessage
	var plaintext, nonce []byte
	var err error

	if err := dec.Decode(&msg); err != nil {
		return []byte{0}, err
	}

	nonce, err = rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, msg.RSANonce, lbl)
	if err != nil {
		return []byte{0}, err
	}

	plaintext, err = aesgcm.Open(nil, nonce, msg.Ciphertext, nil)
	if err != nil {
		return []byte{0}, err
	}

	return plaintext, nil
}
