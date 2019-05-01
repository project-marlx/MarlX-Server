// Package client contains data structures
// used for client handling.
package client

import (
	"crypto/cipher"
	"crypto/rsa"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/MattMoony/MarlX-Server/socks"
)

// Client keeps information about
// a MarlX-Client and the encrypted
// connection to it.
type Client struct {
	ActionQueue []socks.MarlXActionCommand
	AESGCM      cipher.AEAD
	AESKey      []byte
	Conn        *net.TCPConn
	Decoder     *gob.Decoder
	Encoder     *gob.Encoder
	PublicKey   rsa.PublicKey
	Token       []byte
}

// Returns the string representation of a Client
// instance.
func (c *Client) String() string {
	return fmt.Sprintf("Client{ActionQueue: [...], AESGCM: cipher.AEAD{}, AESKey: %x, Conn: *net.TCPConn{}, "+
		"Decoder: *gob.Decoder{}, Encoder: *gob.Encoder{}, PublicKey: rsa.PublicKey{%d, %d}, Token: %x}", c.AESKey,
		c.PublicKey.E, c.PublicKey.N, c.Token)
}

// HasAuthenticated returns whether or
// not the client has authenticated /
// sent its authentication token and
// its integrity has been confirmed.
func (c *Client) HasAuthenticated() bool {
	return len(c.Token) > 0
}

// PushActionQueue is an alias for:
// c.PushActionWithBody(code, nil)
func (c *Client) PushAction(code uint8) {
	c.PushActionWithBody(code, nil)
}

// PushActionWithBody is an alias for:
// PushActionWithBodyAndCallback(code, body, func (str string) {})
func (c *Client) PushActionWithBody(code uint8, body []byte) {
	// c.PushActionWithBodyAndCallback(code, body, func(str string) {})
	c.ActionQueue = append(c.ActionQueue, socks.MarlXActionCommand{code, body})
}

// PushActionWithCallback is an alias for:
// PushActionWithBodyAndCallback(code, nil, cb)
// func (c *Client) PushActionWithCallback(code uint8, cb socks.Callback) {
// 	c.PushActionWithBodyAndCallback(code, nil, cb)
// }

// PushActionWithBodyAndCallback creates a
// socks.MarlXActionCommand with the given parameters
// and appends it to the Client's ActionQueue.
// func (c *Client) PushActionWithBodyAndCallback(code uint8, body []byte, cb socks.Callback) {
// 	c.ActionQueue = append(c.ActionQueue, socks.MarlXActionCommand{code, body, cb})
// }

// ClientLeftError is an error that can
// be thrown if the client leaves during
// TCP Listening.
type ClientLeftError struct {
	Msg string
}

// Returns the string representation of
// a ClientLeftError instance.
// Error: [string]
func (cle *ClientLeftError) Error() string {
	return "Error: " + cle.Msg
}
