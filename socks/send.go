package socks

import (
	"crypto"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"

	"github.com/MattMoony/MarlX-Server/crypto/AESWrapper"
)

// SendRSAPublicKey sends the PublicKey it was handed
// via the given gob.Encoder.
// If an error occurs it will be returned, otherwise the
// return value will be nil.
func SendRSAPublicKey(enc *gob.Encoder, publ rsa.PublicKey) error {
	return enc.Encode(publ)
}

// SendRSAMessage is an alias for:
// SendRSAMessageWithLabel(enc, priv, publ, msg, []byte(""))
func SendRSAMessage(enc *gob.Encoder, priv *rsa.PrivateKey, publ *rsa.PublicKey, msg []byte) error {
	return SendRSAMessageWithLabel(enc, priv, publ, msg, []byte(""))
}

// SendRSAMessageWithLabel encrypts the given msg with the rsa.PublicKey
// publ, signs it with the given rsa.PrivateKey priv and sends it using
// the gob.Encoder that was handed to it.
// If an error occurs it will be returned, otherwise the
// return value will be nil.
func SendRSAMessageWithLabel(enc *gob.Encoder, priv *rsa.PrivateKey, publ *rsa.PublicKey, msg []byte, lbl []byte) error {
	var rsa_msg RSAMessage
	var sha_hash [32]byte
	var err error

	rsa_msg.Ciphertext, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, publ, msg, lbl)
	if err != nil {
		return err
	}

	sha_hash = sha256.Sum256(msg)
	rsa_msg.Signature, err = rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sha_hash[:])
	if err != nil {
		return err
	}

	return enc.Encode(rsa_msg)
}

// SendAESMessage is an alias for:
// SendAESMessageWithLabel(enc, aesgcm, publ, msg, []byte(""))
func SendAESMessage(enc *gob.Encoder, aesgcm cipher.AEAD, publ *rsa.PublicKey, msg []byte) error {
	return SendAESMessageWithLabel(enc, aesgcm, publ, msg, []byte(""))
}

// SendAESMessageWithLabel encrypts the given msg using the given
// cipher.AEAD aesgcm, and sends it, together with the created
// nonce (that was RSA encrypted) using the gob.Encoder enc.
// If an error occurs it will be returned, otherwise the
// return value will be nil.
func SendAESMessageWithLabel(enc *gob.Encoder, aesgcm cipher.AEAD, publ *rsa.PublicKey, msg []byte, lbl []byte) error {
	var aes_msg AESMessage
	var nonce []byte
	var err error

	nonce = AESWrapper.GenerateNonce()
	aes_msg.RSANonce, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, publ, nonce, lbl)
	if err != nil {
		return err
	}

	aes_msg.Ciphertext = aesgcm.Seal(nil, nonce, msg, nil)
	return enc.Encode(aes_msg)
}
