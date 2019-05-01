// Package files provides the file handling
// functionalities in the MarlX-Project.
package files

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"

	"encoding/hex"
	"encoding/json"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"

	"github.com/MattMoony/MarlX-Server/db"
	client_lib "github.com/MattMoony/MarlX-Server/marlx/client"
	"github.com/MattMoony/MarlX-Server/socks"
)

// MTU (Maximum Transmission Unit) is the
// amount of bytes that can maximally be
// transferred in one fragment.
const MTU int64 = int64(128000000)

func CreateFile(dbctx context.Context, dbclient *mongo.Client, u_token []byte, parent_dir string, filename string, size int64,
	asize int64, MIMEType string, salt string, con_clients map[string]*client_lib.Client) (socks.FileInfoHeader, db.StoredClient, error) {
	files := dbclient.Database("marlx").Collection("files")
	users := dbclient.Database("marlx").Collection("users")
	clients := dbclient.Database("marlx").Collection("clients")

	var fih socks.FileInfoHeader
	fih.UniqueId = make([]byte, 32)
	var tempFile db.StoredFile

	var cu_client db.StoredClient
	var fr_client db.StoredClient

FreeTokenLoop:
	for {
		_, err := rand.Read(fih.UniqueId)
		if err != nil {
			return fih, fr_client, err
		}

		err = files.FindOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + fmt.Sprintf("%x", fih.UniqueId)}).Decode(&tempFile)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				break FreeTokenLoop
			}
			log.Println(err.Error())
		}
	}

	fih.Name = filename
	fih.Size = size
	fih.UserToken = u_token
	fih.ParentDir = parent_dir

	var user db.StoredUser

	err := users.FindOne(dbctx, bson.M{"token": u_token}).Decode(&user)
	if err != nil {
		return fih, fr_client, err
	}

	fr_client.FreeBytes = 0

	for _, ct := range user.Clients {
		clients.FindOne(dbctx, bson.M{"token": ct}).Decode(&cu_client)

		_, c_exists := con_clients[fmt.Sprintf("%x", cu_client.Token)]

		if cu_client.FreeBytes > fr_client.FreeBytes && c_exists {
			fr_client = cu_client
		}
	}

	if fr_client.FreeBytes == 0 {
		return fih, fr_client, errors.New("No clients with free space available.")
	}

	files.FindOneAndUpdate(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + parent_dir},
		bson.M{"$addToSet": bson.M{"dirContent": fih.UniqueId}})
	files.InsertOne(dbctx, bson.M{
		"uniqueId":     fmt.Sprintf("%x", u_token) + "_" + fmt.Sprintf("%x", fih.UniqueId),
		"name":         filename,
		"size":         size,
		"actualSize":   asize,
		"MIMEType":     MIMEType,
		"salt":         salt,
		"cTokens":      [][]byte{fr_client.Token},
		"parentDir":    parent_dir,
		"isDir":        false,
		"dirContent":   make([]string, 0),
		"creationTime": time.Now()})

	return fih, fr_client, nil
}

// StoreFile stores a file on the specified client ...
func StoreFile(client *client_lib.Client, file_location string, MTU int, fih socks.FileInfoHeader, dbctx context.Context, dbclient *mongo.Client) {
	f, err := os.Open(file_location)
	if err != nil {
		log.Println(err.Error())
		return
	}

	temp := make([]byte, MTU)

	enc, err := json.Marshal(fih)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var ac socks.MarlXActionCommand

	ac.Action = socks.ACTION_STORE_FILE_HEADER
	ac.Body = enc

	enc, err = json.Marshal(ac)
	if err != nil {
		log.Println(err.Error())
		return
	}

	socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, enc)

	var ff socks.FileFragment

	for i := int32(0); i < fih.FragCount; i++ {
		_, err := f.Read(temp)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		ac.Action = socks.ACTION_STORE_FILE_CONTENT
		ff.Content = temp

		enc, err = json.Marshal(ff)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		ac.Body = enc

		enc, err = json.Marshal(ac)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, enc)
	}
	f.Close()
	log.Printf("Stored file '%s' on client ... ", fih.Name)
}

// StoreFormFile stores a file sent via a multipart/form-data request
// on the specified client ...
func StoreFormFile(client *client_lib.Client, f multipart.File, MTU int64, fih socks.FileInfoHeader, dbctx context.Context,
	dbclient *mongo.Client) error {
	temp := make([]byte, MTU)

	enc, err := json.Marshal(fih)
	if err != nil {
		log.Println(err.Error())
		return errors.New(err.Error())
	}

	var ac socks.MarlXActionCommand

	ac.Action = socks.ACTION_STORE_FILE_HEADER
	ac.Body = enc

	enc, err = json.Marshal(ac)
	if err != nil {
		log.Println(err.Error())
		return errors.New(err.Error())
	}

	socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, enc)

	var ff socks.FileFragment

	for i := int32(0); i < fih.FragCount; i++ {
		if i == fih.FragCount-1 {
			temp = make([]byte, fih.Size-int64(i*fih.FragCount))
		}

		_, err := f.Read(temp)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		ac.Action = socks.ACTION_STORE_FILE_CONTENT
		ff.Content = temp

		enc, err = json.Marshal(ff)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		ac.Body = enc

		enc, err = json.Marshal(ac)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, enc)
	}
	f.Close()
	log.Printf("Stored file '%s' on client ... ", fih.Name)

	return nil
}

// ReceiveFile retrieves a file from the client
// and pipes it to the http-response ...
func ReceiveFile(client *client_lib.Client, priv *rsa.PrivateKey, sf db.StoredFile, u_token []byte, uniqueId []byte,
	fname string, res http.ResponseWriter, streams map[string]socks.WebStream, streams_mutex sync.RWMutex) {

	var ac socks.MarlXActionCommand
	ac.Action = socks.ACTION_REQUEST_FILE

	var rfi socks.RequestedFileInfo
	rfi.UniqueId = uniqueId
	rfi.UserToken = u_token

	rfi.StreamToken = fmt.Sprintf("%x", u_token) + "_" + fmt.Sprintf("%x", uniqueId)
	i := 0

UniqueStreamTokenLoop:
	for {
		streams_mutex.RLock()
		_, exsts := streams[rfi.StreamToken+"#"+string(i)]
		streams_mutex.RUnlock()

		if !exsts {
			break UniqueStreamTokenLoop
		}
		i++
	}

	rfi.StreamToken += "#" + string(i)

	var ws socks.WebStream
	ws.ResW = res

	streams_mutex.Lock()
	streams[rfi.StreamToken] = ws
	streams_mutex.Unlock()

	encb, err := json.Marshal(rfi)
	if err != nil {
		log.Println(err.Error())
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	ac.Body = encb

	encb, err = json.Marshal(ac)
	if err != nil {
		log.Println(err.Error())
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, encb)

	// res.Header().Set("Content-Disposition", "attachment; filename=\""+fname+"\"")

	// wait for async streams to finish ...
	streams_mutex.RLock()
	_, wait := streams[rfi.StreamToken]
	for wait {
		streams_mutex.RUnlock()
		time.Sleep(5 * time.Millisecond)
		streams_mutex.RLock()
		_, wait = streams[rfi.StreamToken]
	}
	streams_mutex.RUnlock()

	log.Printf("Fully sent file '%s' ... ", fname)
}

// DeleteFile tells the specified client
// to remove the file identified by its
// UniqueId from its disk.
func DeleteFile(client *client_lib.Client, u_id string) {
	var dreq socks.DeleteRequest
	dreq.UniqueId = u_id

	encb, err := json.Marshal(dreq)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var ac socks.MarlXActionCommand
	ac.Action = socks.ACTION_DELETE_FILE
	ac.Body = encb

	encb, err = json.Marshal(ac)
	if err != nil {
		log.Println(err.Error())
		return
	}

	socks.SendAESMessage(client.Encoder, client.AESGCM, &client.PublicKey, encb)
}

// MoveFile can be used to move
// a file from one directory to another ...
func MoveFile(dbctx context.Context, dbclient *mongo.Client, u_token []byte, f_id string, targ_dir string) error {
	files := dbclient.Database("marlx").Collection("files")

	var tempF db.StoredFile

	err := files.FindOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + f_id}).Decode(&tempF)
	if err != nil {
		return err
	}

	b_id, err := hex.DecodeString(f_id)
	if err != nil {
		return err
	}

	_, err = files.UpdateOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + tempF.ParentDir},
		bson.M{"$pull": bson.M{"dirContent": b_id}})
	if err != nil {
		return err
	}

	_, err = files.UpdateOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + targ_dir},
		bson.M{"$addToSet": bson.M{"dirContent": b_id}})
	if err != nil {
		return err
	}

	_, err = files.UpdateOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + f_id},
		bson.M{"$set": bson.M{"parentDir": targ_dir}})
	return err
}

// RootDir returns the name
// of the root directory in which
// the given file is located ...
func RootDir(dbctx context.Context, dbclient *mongo.Client, u_token []byte, f_id string) (db.StoredFile, error) {
	files := dbclient.Database("marlx").Collection("files")

	var tempF db.StoredFile
	err := files.FindOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + f_id}).Decode(&tempF)
	if err != nil {
		return tempF, err
	}

	for tempF.ParentDir != "" {
		err = files.FindOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + tempF.ParentDir}).Decode(&tempF)
		if err != nil {
			return tempF, err
		}
	}

	return tempF, nil
}

// FRemoveFiles removes the
// given file completely.
// If the given file is a
// directory, all of its contents
// will be removed as well ...
func FRemoveFiles(dbctx context.Context, files_col *mongo.Collection, con_clients map[string]*client_lib.Client,
	u_token []byte, f_id string) error {
	var tempF db.StoredFile
	err := files_col.FindOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + f_id}).Decode(&tempF)
	if err != nil {
		return err
	}

	for _, ct := range tempF.CTokens {
		tc, connected := con_clients[fmt.Sprintf("%x", ct)]
		if !connected {
			// TODO: HANDLE CLIENT
			// THAT'S CURRENTLY NOT CONNECTED ...
			continue
		}
		DeleteFile(tc, fmt.Sprintf("%x", u_token)+"_"+f_id)
	}

	for _, bfid := range tempF.DirContent {
		err = FRemoveFiles(dbctx, files_col, con_clients, u_token, fmt.Sprintf("%x", bfid))
		if err != nil {
			log.Println("SubFile-Error: " + err.Error())
		}
	}

	b_id, err := hex.DecodeString(f_id)
	if err != nil {
		return err
	}

	files_col.UpdateOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + tempF.ParentDir},
		bson.M{"$pull": bson.M{"dirContent": b_id}})
	_, err = files_col.DeleteOne(dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + f_id})
	if err != nil {
		return err
	}

	return nil
}
