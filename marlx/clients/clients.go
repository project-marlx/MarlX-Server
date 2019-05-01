// Package clients provides functions for operation
// with MarlX-Clients.
package clients

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"

	"encoding/gob"
	"encoding/json"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"

	"github.com/MattMoony/MarlX-Server/db"
	client_lib "github.com/MattMoony/MarlX-Server/marlx/client"
	"github.com/MattMoony/MarlX-Server/socks"
)

// HandleClient provides the main logic behind
// client handling. It should be called immediately
// when a TCP-Connection has been established.
// (It should probably be a separate Goroutine if one
// wants to handle multiple clients without blocking)
func HandleClient(conn *net.TCPConn, priv *rsa.PrivateKey, con_clients map[string]*client_lib.Client, dbctx context.Context,
	dbclient *mongo.Client, streams map[string]socks.WebStream, streams_mutex sync.RWMutex) {
	var client client_lib.Client
	var block cipher.Block
	var err error

	client.Conn = conn
	client.Encoder = gob.NewEncoder(conn)
	client.Decoder = gob.NewDecoder(conn)

	err = socks.RSAKeyExchange(client.Encoder, client.Decoder, priv, &client.PublicKey)
	if err != nil {
		log.Println(err.Error())
		return
	}

	client.AESKey, err = socks.ReceiveRSAMessage(client.Decoder, priv, &client.PublicKey)
	if err != nil {
		log.Println(err.Error())
		return
	}

	block, err = aes.NewCipher(client.AESKey)
	if err != nil {
		log.Println(err.Error())
		return
	}

	client.AESGCM, err = cipher.NewGCM(block)
	if err != nil {
		log.Println(err.Error())
		return
	}

	client.PushAction(socks.ACTION_IDENTIFY)

	go HandleClientIn(&client, priv, con_clients, dbctx, dbclient, streams, streams_mutex)
	go HandleClientOut(&client, priv, con_clients, dbctx, dbclient)
}

// HandleClientIn handles the input stream
// of the given client.
// In order to run it at the same time as
// HandleClientOut, it should be a separate
// Goroutine.
func HandleClientIn(client *client_lib.Client, priv *rsa.PrivateKey, con_clients map[string]*client_lib.Client, dbctx context.Context,
	dbclient *mongo.Client, streams map[string]socks.WebStream, streams_mutex sync.RWMutex) {
	var actionCommand socks.MarlXActionCommand

	var frh socks.FileResponseHeader
	var ff socks.FileFragment
	var diup socks.DiskinfoUpdate

	var plainmsg []byte
	var err error

AuthenticationLoop:
	for !client.HasAuthenticated() {
		err = ReceiveAuthentication(client, priv, dbctx, dbclient)
		if err != nil {
			switch err.(type) {
			case *client_lib.ClientLeftError:
				client.PushAction(socks.ACTION_CLOSE_SOCKET)
				return
			default:
				switch err {
				case mongo.ErrNoDocuments:
					client.PushActionWithBody(socks.ACTION_ERROR, []byte("Unknown client!"))
					continue AuthenticationLoop
				default:
					client.PushActionWithBody(socks.ACTION_ERROR, []byte(err.Error()))
					continue AuthenticationLoop
				}
			}
		}

		break
	}

	log.Println("Client has authenticated!")
	con_clients[fmt.Sprintf("%x", client.Token)] = client

InputLoop:
	for {
		// Receive AES-encrypted message from the client ...
		plainmsg, err = socks.ReceiveAESMessage(client.Decoder, client.AESGCM, priv)
		if err != nil {
			log.Println("client left")
			client.PushAction(socks.ACTION_CLOSE_SOCKET)
			return
		}

		// check if plainmsg is not empty ...
		if plainmsg == nil || len(plainmsg) == 0 {
			client.PushAction(socks.ACTION_CLOSE_SOCKET)
			return
		}

		// creates actionCommand from plainmsg ...
		err = json.Unmarshal(plainmsg, &actionCommand)
		if err != nil {
			log.Println(err.Error())
			continue InputLoop
		}

		// callback_str := ""

		// switch through the actionCommand's options ...
		switch actionCommand.Action {
		// in case it's an 'identification' packet ...
		case socks.ACTION_TOKEN_IDENTIFICATION:
			err = CheckToken(actionCommand.Body, dbctx, dbclient)
			if err != nil {
				client.PushActionWithBody(socks.ACTION_ERROR, []byte("Unknown client!"))
				continue InputLoop
			}

			client.Token = actionCommand.Body
		// in case it's a 'disk-info-update' packet ...
		case socks.ACTION_DISKINFO_UPDATE:
			err = json.Unmarshal(actionCommand.Body, &diup)
			if err != nil {
				log.Println(err.Error())
				continue InputLoop
			}

			log.Println("received diskinfo update")

			UpdateDiskinfo(client, diup, dbctx, dbclient)
		case socks.ACTION_RESPOND_FILE_HEADER:
			err = json.Unmarshal(actionCommand.Body, &frh)
			if err != nil {
				log.Println(err.Error())
				continue InputLoop
			}

			streams_mutex.RLock()
			ws, exs := streams[frh.StreamToken]
			streams_mutex.RUnlock()

			if !exs {
				log.Println("Stream doesn't exist ... ")
				continue InputLoop
			}

			ws.MTU = frh.MTU
			ws.Size = frh.Size
		case socks.ACTION_RESPOND_FILE_CONTENT:
			err = json.Unmarshal(actionCommand.Body, &ff)
			if err != nil {
				log.Println(err.Error())
				continue InputLoop
			}

			streams_mutex.RLock()
			ws, exs := streams[ff.StreamToken]
			streams_mutex.RUnlock()

			if !exs {
				log.Println("Stream doesn't exist ... ")
				continue InputLoop
			}

			ws.ResW.Write(ff.Content)

			if ff.Index == ff.Total-1 {
				streams_mutex.Lock()
				delete(streams, ff.StreamToken)
				streams_mutex.Unlock()
			}
		default:
			// log.Print("default: ")
			// log.Println(actionCommand.Action)
		}

		// actionCommand.Callback(callback_str)
	}
}

// HandleClientOut handles the output stream
// of the given client.
// In order to run it at the same time as
// HandleClientOut, it should be a separate
// Goroutine.
func HandleClientOut(client *client_lib.Client, priv *rsa.PrivateKey, con_clients map[string]*client_lib.Client, dbctx context.Context,
	dbclient *mongo.Client) {

	for {
		for _, action := range client.ActionQueue {
			// callback_str := ""

			switch action.Action {
			case socks.ACTION_CLOSE_SOCKET:
				err := client.Conn.Close()
				if err != nil {
					log.Println(err.Error())
				}

				delete(con_clients, fmt.Sprintf("%x", client.Token))
				return
			default:
				enc, err := json.Marshal(action)
				if err != nil {
					log.Println(err.Error())
					break
				}

				socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, enc)
				return
			}

			// action.Callback(callback_str)
		}
	}
}

// ReceiveAuthentication waits for the client
// to send a "001 Token-Identification" message
// and confirms the integrity of the received token.
// Returns: error (if any occured)
func ReceiveAuthentication(client *client_lib.Client, priv *rsa.PrivateKey, dbctx context.Context, dbclient *mongo.Client) error {
	var command socks.MarlXActionCommand
	var plainmsg, token []byte
	var err error

	plainmsg, err = socks.ReceiveAESMessage(client.Decoder, client.AESGCM, priv)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if plainmsg == nil || len(plainmsg) == 0 {
		client.PushAction(socks.ACTION_CLOSE_SOCKET)
		return &client_lib.ClientLeftError{"Client left!"}
	}

	err = json.Unmarshal(plainmsg, &command)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if command.Action != 1 {
		return errors.New("Please identify first!")
	}
	token = command.Body

	err = CheckToken(token, dbctx, dbclient)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	client.Token = token
	return nil
}

func CheckToken(token []byte, dbctx context.Context, dbclient *mongo.Client) error {
	var fetched_client db.StoredClient

	stored_clients := dbclient.Database("marlx").Collection("clients")
	res := stored_clients.FindOne(dbctx, bson.M{"token": token})

	err := res.Decode(&fetched_client)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDiskinfo(client *client_lib.Client, du socks.DiskinfoUpdate, dbctx context.Context, dbclient *mongo.Client) {
	clients := dbclient.Database("marlx").Collection("clients")
	clients.FindOneAndUpdate(dbctx, bson.M{"token": client.Token}, bson.M{"$set": bson.M{"freeBytes": du.FreeBytes, "totalBytes": du.TotalBytes, "hostname": du.Hostname, "MTU": du.MTU}})
}
