// Package socks is used for socket communication
// in the MarlX project.
package socks

import (
	"crypto/rsa"
	"encoding/gob"
	"fmt"
	"net/http"
)

// RSAMessage describes a message encrypted and
// signed with RSA.
type RSAMessage struct {
	Ciphertext []byte
	Signature  []byte
}

// Returns the string representation of the
// RSAMessage object.
// RSAMessage{Ciphertext: [hex], Signature: [hex]}
func (msg *RSAMessage) String() string {
	return fmt.Sprintf("RSAMessage{Ciphertext: %x, Signature %x}", msg.Ciphertext, msg.Signature)
}

// AESMessage describes an AES-encrypted message
// plus its Nonce.
type AESMessage struct {
	Ciphertext []byte
	RSANonce   []byte
}

// Returns the string representation of the
// AESMessage object.
// AESMessage{Ciphertext: [hex], Signature: [hex]}
func (msg *AESMessage) String() string {
	return fmt.Sprintf("AESMessage{Ciphertext: %x, RSANonce: %x}", msg.Ciphertext, msg.RSANonce)
}

type Callback func(string)

// MarlXActionCommand is used to transfer Action-Commands
// between client and server.
type MarlXActionCommand struct {
	Action uint8
	Body   []byte
	// Callback Callback
}

// Returns the string representation of the
// MarlXActionCommand instance.
// MarlXActionCommand{Action: [dec]}
func (msg *MarlXActionCommand) String() string {
	return fmt.Sprintf("MarlXActionCommand{Action: %03d}", msg.Action)
}

// 0XX - CLIENT MESSAGES

// ACTION_TOKEN_IDENTIFICATION is the code that
// will be used, if a client sends its token
// in order to identify itself.
const ACTION_TOKEN_IDENTIFICATION uint8 = 1

// ACTION_DISKINFO_UPDATE sends diskinformation
// from a client to a server.
const ACTION_DISKINFO_UPDATE uint8 = 2

// ACTION_RESPOND_FILE_HEADER is sent from a client to respond
// to an ACTION_REQUEST_FILE_HEADER code. It contains the
// requested file, and will be forwarded by the
// server.
const ACTION_RESPOND_FILE_HEADER uint8 = 3

// ACTION_RESPOND_FILE_CONTENT transmits the content of
// a file who's information was previously sent via
// a ACTION_RESPOND_FILE_HEADER-Code message.
const ACTION_RESPOND_FILE_CONTENT uint8 = 4

// 1XX - (CLIENT/SERVER)-MESSAGES

// ACTION_INFORMATION can be used to send information
// between client and server.
const ACTION_INFORMATION uint8 = 100

// ACTION_ERROR can be used to send error-messages between
// client and server.
const ACTION_ERROR uint8 = 101

// 2XX - SERVER MESSAGES

// ACTION_IDENTIFY is a demand made by the server
// that tells the client to respond with a
// ACTION_TOKEN_IDENTIFICATION-Code message.
const ACTION_IDENTIFY uint8 = 200

// ACTION_UPDATE_DISKINFO requests a diskinfo
// update from the client.
const ACTION_UPDATE_DISKINFO uint8 = 202

// ACTION_STORE_FILE_HEADER tells a client to store the file,
// and tells the server to find a suitable client
// which can store the specified file.
const ACTION_STORE_FILE_HEADER uint8 = 203

// ACTION_STORE_FILE_CONTENT transmits the content of
// a file who's information was previously sent via
// a ACTION_STORE_FILE_HEADER-Code message.
const ACTION_STORE_FILE_CONTENT uint8 = 204

// ACTION_REQUEST_FILE tells a client to respond
// with the specified file, if sent to a server
// it will forward the message to the client that
// stored the file.
const ACTION_REQUEST_FILE uint8 = 205

// ACTION_DELETE_FILE tells a client to remove
// the specified file from its disk.
const ACTION_DELETE_FILE uint8 = 206

// ACTION_CLOSE_SOCKET tells the server to close
// the socket.
const ACTION_CLOSE_SOCKET uint8 = 255

// FileInfoHeader contains information about
// a file to-be-transferred.
type FileInfoHeader struct {
	FragCount int32
	Name      string
	ParentDir string
	Size      int64
	UniqueId  []byte
	UserToken []byte
}

// Returns the string representation of a
// FileInfoHeader instance.
func (fih *FileInfoHeader) String() string {
	return fmt.Sprintf("FileInfoHeader{FragCount: %d, Name: %s, Size: %d, UniqueId: %x, UserToken: %x}",
		fih.FragCount, fih.Name, fih.Size, fih.UniqueId, fih.UserToken)
}

// RequestedFileInfo contains info about
// a file to-be-requested.
type RequestedFileInfo struct {
	UniqueId    []byte
	UserToken   []byte
	StreamToken string
}

// FileResponseHeader contains info about
// a file respond to-be-sent.
type FileResponseHeader struct {
	MTU         int64
	Size        int64
	StreamToken string
}

// FileFragment contains the Unique-Id + a
// fragment of a file being transferred.
type FileFragment struct {
	StreamToken string
	Index       int64
	Total       int64
	Content     []byte
}

// DiskinfoUpdate contains information
// about the storage situation of a client.
type DiskinfoUpdate struct {
	Hostname   string
	MTU        uint64
	FreeBytes  uint64
	TotalBytes uint64
}

// WebStream helps the file-fragments
// to be sent to the correct web-user.
type WebStream struct {
	ResW http.ResponseWriter
	MTU  int64
	Size int64
}

// DeleteRequest contains information
// about a file to-be-removed.
type DeleteRequest struct {
	UniqueId string
}

// RSAKeyExchange can be used for the initial
// key exchange between an RSA client & an
// RSA server.
// It returns an error if any occurs.
func RSAKeyExchange(enc *gob.Encoder, dec *gob.Decoder, priv *rsa.PrivateKey, publ *rsa.PublicKey) error {
	err := ReceiveRSAPublicKey(dec, publ)
	if err != nil {
		return err
	}

	err = SendRSAPublicKey(enc, priv.PublicKey)
	if err != nil {
		return err
	}

	return nil
}
