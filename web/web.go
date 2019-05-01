// Package web controls the MarlX
// web server functionality.
package web

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"

	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"

	"compress/gzip"
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/MattMoony/MarlX-Server/db"
	client_lib "github.com/MattMoony/MarlX-Server/marlx/client"
	marlx_files "github.com/MattMoony/MarlX-Server/marlx/files"
	"github.com/MattMoony/MarlX-Server/socks"
)

// PORT defines the portnumber on which
// the web server will be listening.
const PORT = 80

// glob_dbctx is the global database
// context.
var glob_dbctx context.Context

// glob_dbclient is the global MongoDB
// client.
var glob_dbclient *mongo.Client

// glob_con_clients is a map consisting
// of all currently connected clients ...
var glob_con_clients map[string]*client_lib.Client

// glob_priv_key contains the server's
// rsa private key ...
var glob_priv_key *rsa.PrivateKey

// glob_streams contains streams
// for the receiving of files
var glob_streams map[string]socks.WebStream

// glob_streams_mutex prevents
// simultaneous read/write operations
var glob_streams_mutex sync.RWMutex

// glob_sessions maps browser tokens
// to marlx user-tokens ...
var glob_sessions *sessions.CookieStore

// SignupRequest keeps information
// about a new user.
type SignUpRequest struct {
	Username string
	Email    string
	PwdHash  string
}

// LoginRequest keeps information
// about a user wanting to sign-in.
type LoginRequest struct {
	AuthType string
	Email    string
	Username string
	Password string
}

// NewDirRequest keeps information
// about a new directory ...
type NewDirRequest struct {
	DirName   string
	ParentDir string
}

// RenameRequest keeps information
// about a file to-be-renamed ...
type RenameRequest struct {
	UniqueId string
	NewName  string
}

// DeleteRequest keeps information
// about a file to-be-deleted ...
type DeleteRequest struct {
	UniqueId string
}

// RecoverRequest keeps information
// about a file to-be-recovered ...
type RecoverRequest struct {
	UniqueId string
}

// MoveRequest keeps information
// about a file to-be-moved ...
type MoveRequest struct {
	UniqueId string
	TargDir  string
}

// RecaptchaRequest contains a
// recaptcha token.
type RecaptchaRequest struct {
	Token string
}

// RecaptchaAPIRequest contains
// info, which will be sent
// directely to the google-servers.
type RecaptchaAPIRequest struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
	Remoteip string `json:"remoteip"`
}

func (rar *RecaptchaAPIRequest) URLEncode() []byte {
	return []byte(fmt.Sprintf("secret=%s&response=%s&remoteip=%s", rar.Secret, rar.Response, rar.Remoteip))
}

func EnableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Credentials", "true")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, "+
		"X-CSRF-Token, Authorization")
}

type gzResWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzResWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func RequestHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		EnableCORS(&res)

		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			res.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(res)
			defer gz.Close()
			fn(gzResWriter{Writer: gz, ResponseWriter: res}, req)
		} else {
			fn(res, req)
		}

	}
}

// RecaptchaHandler handles recaptcha
// events from the frontend.
func RecaptchaHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	dec := json.NewDecoder(req.Body)
	var rreq RecaptchaRequest

	err := dec.Decode(&rreq)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	rareq := RecaptchaAPIRequest{"6LfaipMUAAAAAMumS-sD2PHslaOG7-YTQr1ajdxW", rreq.Token, req.RemoteAddr}

	resp, err := http.Post("https://www.google.com/recaptcha/api/siteverify", "application/x-www-form-urlencoded", bytes.NewBuffer(rareq.URLEncode()))
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	res.Write(body)
}

// SignUpHandler handles sign-up requests
// from the frontend.
func SignUpHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	dec := json.NewDecoder(req.Body)
	var sreq SignUpRequest

	err := dec.Decode(&sreq)
	if err != nil {
		log.Println(err)
		http.Error(res, "{\"success\": false, \"error\": \"Bad JSON!\", \"erroneous_attr\": \"JSON\"}", http.StatusBadRequest)
		return
	}

	log.Println(sreq)

	token := make([]byte, 32)
	users := glob_dbclient.Database("marlx").Collection("users")
	files := glob_dbclient.Database("marlx").Collection("files")
	var tempUser struct{}

	err = users.FindOne(glob_dbctx, bson.M{"username": sreq.Username}).Decode(&tempUser)
	if err == nil {
		res.Write([]byte("{\"success\": false, \"error\": \"Username already taken!\", \"erroneous_attr\": \"username\"}"))
		return
	}

	err = users.FindOne(glob_dbctx, bson.M{"email": sreq.Email}).Decode(&tempUser)
	if err == nil {
		res.Write([]byte("{\"success\": false, \"error\": \"Email already taken!\", \"erroneous_attr\": \"email\"}"))
		return
	}

FreeTokenLoop:
	for {
		_, err = rand.Read(token)
		if err != nil {
			res.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		err := users.FindOne(glob_dbctx, bson.M{"token": token}).Decode(&tempUser)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				break FreeTokenLoop
			}
			log.Println(err.Error())
		}
	}

	users.InsertOne(glob_dbctx, bson.M{
		"token":    token,
		"email":    sreq.Email,
		"username": sreq.Username,
		"password": sreq.PwdHash,
		"clients":  make([][]byte, 0)})
	files.InsertOne(glob_dbctx, bson.M{
		"name":         "root",
		"uniqueId":     fmt.Sprintf("%x", token) + "_root",
		"size":         0,
		"actualSize":   0,
		"MIMEType":     "text/directory",
		"salt":         "",
		"cTokens":      make([][]byte, 0),
		"parentDir":    "",
		"isDir":        true,
		"dirContent":   make([][]byte, 0),
		"creationTime": time.Now()})
	files.InsertOne(glob_dbctx, bson.M{
		"name":         "trash",
		"uniqueId":     fmt.Sprintf("%x", token) + "_trash",
		"size":         0,
		"actualSize":   0,
		"MIMEType":     "text/directory",
		"salt":         "",
		"cTokens":      make([][]byte, 0),
		"parentDir":    "",
		"isDir":        true,
		"dirContent":   make([][]byte, 0),
		"creationTime": time.Now()})

	b_token := make([]byte, 32)

	_, err = rand.Read(b_token)
	if err != nil {
		res.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	sess, err := glob_sessions.Get(req, fmt.Sprintf("%x", b_token))
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal Error\", \"erroneous_attr\": \"\"}", http.StatusInternalServerError)
		return
	}

	sess.Values["u_token"] = token
	sess.Save(req, res)

	tkn_cookie := http.Cookie{Name: "marlx_tkn", Value: fmt.Sprintf("%x", b_token), Path: "/"}
	http.SetCookie(res, &tkn_cookie)

	res.Write([]byte("{\"success\": true, \"error\": \"\", \"erroneous_attr\": \"\"}"))
}

// LoginHandler handles login requests
// from the frontend.
func LoginHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	dec := json.NewDecoder(req.Body)
	var lreq LoginRequest

	err := dec.Decode(&lreq)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Bad JSON!\", \"erroneous_attr\": \"JSON\"}", http.StatusBadRequest)
		return
	}

	users := glob_dbclient.Database("marlx").Collection("users")
	var tempUser db.StoredUser

	if lreq.AuthType == "email" {
		err := users.FindOne(glob_dbctx, bson.M{"email": lreq.Email}).Decode(&tempUser)
		if err != nil {
			res.Write([]byte("{\"success\": false, \"error\": \"Email unknown!\", \"erroneous_attr\": \"identity\"}"))
			return
		}
	} else {
		err := users.FindOne(glob_dbctx, bson.M{"username": string(lreq.Username)}).Decode(&tempUser)
		if err != nil {
			log.Println(err)
			res.Write([]byte("{\"success\": false, \"error\": \"Username unknown!\", \"erroneous_attr\": \"identity\"}"))
			return
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(tempUser.Password), []byte(lreq.Password))
	if err != nil {
		res.Write([]byte("{\"success\": false, \"error\": \"Wrong password!\", \"erroneous_attr\": \"pwd\"}"))
		return
	}

	b_token := make([]byte, 32)

	_, err = rand.Read(b_token)
	if err != nil {
		res.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		return
	}

	sess, err := glob_sessions.Get(req, fmt.Sprintf("%x", b_token))
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal Error!\", \"erroneous_attr\": \"\"}", http.StatusInternalServerError)
		return
	}

	sess.Values["u_token"] = tempUser.Token
	sess.Save(req, res)

	tkn_cookie := http.Cookie{Name: "marlx_tkn", Value: fmt.Sprintf("%x", b_token), Path: "/"}
	http.SetCookie(res, &tkn_cookie)

	res.Write([]byte("{\"success\": true, \"error\": \"\", \"erroneous_attr\": \"\"}"))
}

// CredentialsHandler gets the user's
// credentials
func CredentialsHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Not logged in!\", \"erroneous_attr\": \"cookie\"}", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal Error\", \"erroneous_attr\": \"\"}", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "{\"success\": false, \"error\": \"Unknown user!\", \"erroneous_attr\": \"user-token\"}", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	users := glob_dbclient.Database("marlx").Collection("users")
	var user db.StoredUser

	err = users.FindOne(glob_dbctx, bson.M{"token": u_token}).Decode(&user)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal user error!\", \"erroneous_attr\": \"user-token\"}", http.StatusInternalServerError)
		return
	}

	res.Write([]byte(`{"username": "` + user.Username + `", "email": "` + user.Email + `"}`))
}

// NewClientHandler handles the
// creation of a new client ...
func NewClientHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Cookie missing!\", \"erroneous_attr\": \"token cookie\"}", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal Error!\", \"erroneous_attr\": \"\"}", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "{\"success\": false, \"error\": \"Unknown user!\", \"erroneous_attr\": \"user-token\"}", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	users := glob_dbclient.Database("marlx").Collection("users")
	clients := glob_dbclient.Database("marlx").Collection("clients")

	var c db.StoredClient
	c.Token = make([]byte, 32)

	var tempClient db.StoredClient

FreeTokenLoop:
	for {
		_, err = rand.Read(c.Token)
		if err != nil {
			res.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		err := clients.FindOne(glob_dbctx, bson.M{"token": c.Token}).Decode(&tempClient)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				break FreeTokenLoop
			}
			log.Println(err.Error())
		}
	}

	c.FreeBytes = 0
	c.TotalBytes = 0
	c.Hostname = ""

	users.FindOneAndUpdate(glob_dbctx, bson.M{"token": u_token}, bson.M{"$addToSet": bson.M{"clients": c.Token}})
	clients.InsertOne(glob_dbctx, c)

	res.Write([]byte(fmt.Sprintf("%x", c.Token)))
}

// ListClientHandler lists all clients
// of the given user ...
func ListClientHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Cookie missing!\", \"erroneous_attr\": \"token cookie\"}", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "\"success\": false, \"error\": \"Internal Error!\", \"erroneous_attr\": \"\"", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "{\"success\": false, \"error\": \"Unknown user!\", \"erroneous_attr\": \"user-token\"}", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	var user db.StoredUser

	users := glob_dbclient.Database("marlx").Collection("users")
	err = users.FindOne(glob_dbctx, bson.M{"token": u_token}).Decode(&user)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal user error!\", \"erroneous_attr\": \"user-token\"}", http.StatusBadRequest)
		return
	}

	cl_tokens := make([]string, len(user.Clients))
	for i, t := range user.Clients {
		cl_tokens[i] = hex.EncodeToString(t)
	}

	encb, err := json.Marshal(cl_tokens)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal JSON error!\", \"erroneous_attr\": \"JSON\"}", http.StatusInternalServerError)
		return
	}

	res.Write(encb)
}

// InfoClientHandler gives information
// about a certain client ...
func InfoClientHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Cookie missing!\", \"erroneous_attr\": \"user cookie\"}", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "{\"success\": false, \"error\": \"Internal session error!\", \"erroneous_attr\": \"\"}", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "{\"success\": false, \"error\": \"Unknown user!\", \"erroneous_attr\": \"user-token\"}", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	in_cl_tkn := req.URL.Query().Get("tkn")
	if in_cl_tkn == "" {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	cl_token, err := hex.DecodeString(in_cl_tkn)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	users := glob_dbclient.Database("marlx").Collection("users")
	clients := glob_dbclient.Database("marlx").Collection("clients")

	var user db.StoredUser
	err = users.FindOne(glob_dbctx, bson.M{"token": u_token}).Decode(&user)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	eq := false
	for _, c := range user.Clients {
		if bytes.Equal(c, cl_token) {
			eq = true
			break
		}
	}

	if !eq {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	var client db.StoredClient
	err = clients.FindOne(glob_dbctx, bson.M{"token": cl_token}).Decode(&client)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	enc, err := json.Marshal(client)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	res.Write(enc)
}

// DirectoryReceiveHandler handles the
// creation of directories ...
func DirectoryReceiveHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "multipart/form-data")
	res.Header().Set("Content-Type", "application/json")

	dec := json.NewDecoder(req.Body)
	var dreq NewDirRequest

	err := dec.Decode(&dreq)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	files := glob_dbclient.Database("marlx").Collection("files")
	dir_unique_id := make([]byte, 32)
	var tempFile db.StoredFile

FreeTokenLoop:
	for {
		_, err := rand.Read(dir_unique_id)
		if err != nil {
			log.Println(err.Error())
			http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + fmt.Sprintf("%x", dir_unique_id)}).Decode(&tempFile)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				break FreeTokenLoop
			}
			log.Println(err.Error())
		}
	}

	files.FindOneAndUpdate(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + dreq.ParentDir},
		bson.M{"$addToSet": bson.M{"dirContent": dir_unique_id}})
	files.InsertOne(glob_dbctx, bson.M{
		"uniqueId":     fmt.Sprintf("%x", u_token) + "_" + fmt.Sprintf("%x", dir_unique_id),
		"name":         dreq.DirName,
		"size":         0,
		"actualSize":   0,
		"MIMEType":     "text/directory",
		"salt":         "",
		"cTokens":      make([][]byte, 0),
		"parentDir":    dreq.ParentDir,
		"isDir":        true,
		"dirContent":   make([][]byte, 0),
		"creationTime": time.Now()})

	res.Write([]byte("{\"success\": true, \"error\": \"\"}"))
}

// FileReceiveHandler handles the
// upload of files ...
func FileReceiveHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "multipart/form-data")
	res.Header().Set("Content-Type", "application/json")

	err := req.ParseMultipartForm(1)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	f, fh, err := req.FormFile("file")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("File-Size: %d\n", fh.Size)

	parent_dir := req.URL.Query().Get("parDir")

	if parent_dir == "" {
		http.Error(res, "Error 400: Bad Request - parent dir", http.StatusBadRequest)
		return
	}

	osize_str := req.URL.Query().Get("osize")

	if osize_str == "" {
		http.Error(res, "Error 400: Bad Request - original size", http.StatusBadRequest)
		return
	}

	osize, err := strconv.ParseInt(osize_str, 10, 64)

	if err != nil {
		http.Error(res, "Error 400: Bad Request - original size format", http.StatusBadRequest)
		return
	}

	salt := req.URL.Query().Get("salt")

	if salt == "" {
		http.Error(res, "Error 400: Bad Request - salt", http.StatusBadRequest)
		return
	}

	mime_type := req.URL.Query().Get("type")

	if mime_type == "" {
		http.Error(res, "Error 400: Bad Request - mime type", http.StatusBadRequest)
		return
	}

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error - token", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	files := glob_dbclient.Database("marlx").Collection("files")

	var p_directory db.StoredFile

	err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + parent_dir}).Decode(&p_directory)
	if err != nil {
		http.Error(res, "Error 400: Bad Request - par dir not exist", http.StatusBadRequest)
		return
	}

	if !p_directory.IsDir {
		http.Error(res, "Error 400: Bad Request - not dir", http.StatusBadRequest)
		return
	}

	fih, fr_client, err := marlx_files.CreateFile(glob_dbctx, glob_dbclient, u_token, parent_dir, fh.Filename, fh.Size, osize, mime_type, salt, glob_con_clients)
	if err != nil {
		res.Write([]byte(fmt.Sprintf("{success: false, error: \"%s\"", err.Error())))
		return
	}

	fih.FragCount = int32(math.Ceil(float64(fih.Size) / float64(fr_client.MTU)))

	err = marlx_files.StoreFormFile(glob_con_clients[fmt.Sprintf("%x", fr_client.Token)], f, fr_client.MTU, fih, glob_dbctx, glob_dbclient)
	if err != nil {
		log.Println(err.Error())
		http.Error(res, "Error 500: Internal Server Error - form file error", http.StatusInternalServerError)
		return
	}

	res.Write([]byte("{success: true, error: \"\"}"))
}

// DirectoryRequestHandler handles directory
// content requests ...
func DirectoryRequestHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	d_id := req.URL.Query().Get("d")

	if d_id == "" {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	files := glob_dbclient.Database("marlx").Collection("files")
	var dir db.StoredFile

	err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + d_id}).Decode(&dir)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	if !dir.IsDir {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	con_ids := dir.DirContent
	items := make([]db.StoredFile, 0)

	var temp_file db.StoredFile

	for _, id := range con_ids {
		err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + fmt.Sprintf("%x", id)}).Decode(&temp_file)
		if err != nil {
			continue
		}

		items = append(items, temp_file)
	}

	encoded, err := json.Marshal(items)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	res.Write(encoded)
}

// FileRequestHandler handles file requests
// from the frontend.
func FileRequestHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")

	u_id := req.URL.Query().Get("u")

	if u_id == "" {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	files := glob_dbclient.Database("marlx").Collection("files")
	var sf db.StoredFile

	err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + u_id}).Decode(&sf)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	var t_cl *client_lib.Client = nil

	for _, c := range sf.CTokens {
		cu_cl, exsts := glob_con_clients[fmt.Sprintf("%x", c)]
		if exsts {
			t_cl = cu_cl
			break
		}
	}

	if t_cl == nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_b_id, err := hex.DecodeString(u_id)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	marlx_files.ReceiveFile(t_cl, glob_priv_key, sf, u_token, u_b_id, sf.Name, res, glob_streams, glob_streams_mutex)
}

func DirTraceHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	f_id := req.URL.Query().Get("u")

	if f_id == "" {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	files := glob_dbclient.Database("marlx").Collection("files")

	var sf db.StoredFile
	err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + f_id}).Decode(&sf)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	par_trace := []db.StoredFile{sf}
	for sf.ParentDir != "" {
		err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + sf.ParentDir}).Decode(&sf)
		if err != nil {
			http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
			return
		}

		par_trace = append(par_trace, sf)
	}

	encb, err := json.Marshal(par_trace)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusBadRequest)
		return
	}

	res.Write(encb)
}

// RenameHandler is used to rename files
// stored in the MongoDB.
func RenameHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	dec := json.NewDecoder(req.Body)
	var rreq RenameRequest

	err = dec.Decode(&rreq)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	files := glob_dbclient.Database("marlx").Collection("files")

	_, err = files.UpdateOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + rreq.UniqueId},
		bson.M{"$set": bson.M{"name": rreq.NewName}})
	if err != nil {
		log.Println(err)
		res.Write([]byte("{\"success\":false, \"error\":\"internal\"}"))
	}

	res.Write([]byte("{\"success\":true, \"error\":\"\"}"))
}

func DeleteItemHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	dec := json.NewDecoder(req.Body)
	var dreq DeleteRequest

	err = dec.Decode(&dreq)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	files := glob_dbclient.Database("marlx").Collection("files")

	var tempF db.StoredFile
	err = files.FindOne(glob_dbctx, bson.M{"uniqueId": fmt.Sprintf("%x", u_token) + "_" + dreq.UniqueId}).Decode(&tempF)
	if err != nil {
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	rd, err := marlx_files.RootDir(glob_dbctx, glob_dbclient, u_token, dreq.UniqueId)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	if strings.Split(rd.UniqueId, "_")[1] == "trash" {
		files := glob_dbclient.Database("marlx").Collection("files")
		marlx_files.FRemoveFiles(glob_dbctx, files, glob_con_clients, u_token, dreq.UniqueId)
	} else {
		marlx_files.MoveFile(glob_dbctx, glob_dbclient, u_token, dreq.UniqueId, "trash")
	}

	res.Write([]byte("{\"success\":true, \"error\":\"\"}"))
}

func MoveItemHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Accept", "application/json")
	res.Header().Set("Content-Type", "application/json")

	ck, err := req.Cookie("marlx_tkn")
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess, err := glob_sessions.Get(req, ck.Value)
	if err != nil {
		http.Error(res, "Error 500: Internal Server Error", http.StatusInternalServerError)
		return
	}

	u_token, exists := sess.Values["u_token"].([]byte)
	if !exists {
		log.Println("nope")
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	sess.Save(req, res)

	dec := json.NewDecoder(req.Body)
	var mreq MoveRequest

	err = dec.Decode(&mreq)
	if err != nil {
		log.Println(err)
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	err = marlx_files.MoveFile(glob_dbctx, glob_dbclient, u_token, mreq.UniqueId, mreq.TargDir)
	if err != nil {
		log.Println(err)
		http.Error(res, "Error 400: Bad Request", http.StatusBadRequest)
		return
	}

	res.Write([]byte("{\"success\":true, \"error\":\"\"}"))
}

// Start should be used to launch the
// web server.
func Start(con_clients map[string]*client_lib.Client, priv *rsa.PrivateKey, dbctx context.Context, dbclient *mongo.Client,
	streams map[string]socks.WebStream, streams_mutex sync.RWMutex) {

	glob_dbctx = dbctx
	glob_dbclient = dbclient

	glob_con_clients = con_clients
	glob_priv_key = priv

	glob_streams = streams
	glob_streams_mutex = streams_mutex

	buff := make([]byte, 32)
	rand.Read(buff)
	glob_sessions = sessions.NewCookieStore(buff)

	__dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc("/api/sign-up", RequestHandler(SignUpHandler))
	http.HandleFunc("/api/login", RequestHandler(LoginHandler))

	http.HandleFunc("/api/recaptcha", RequestHandler(RecaptchaHandler))
	http.HandleFunc("/api/creds", RequestHandler(CredentialsHandler))

	http.HandleFunc("/api/clnts/new", RequestHandler(NewClientHandler))
	http.HandleFunc("/api/clnts/list", RequestHandler(ListClientHandler))
	http.HandleFunc("/api/clnts/info", RequestHandler(InfoClientHandler))

	http.HandleFunc("/api/upd/file", RequestHandler(RenameHandler))

	http.HandleFunc("/api/rec/dir", RequestHandler(DirectoryReceiveHandler))
	http.HandleFunc("/api/rec/file", RequestHandler(FileReceiveHandler))

	http.HandleFunc("/api/req/dir", RequestHandler(DirectoryRequestHandler))
	http.HandleFunc("/api/req/file", RequestHandler(FileRequestHandler))
	http.HandleFunc("/api/req/trace", RequestHandler(DirTraceHandler))

	http.HandleFunc("/api/del/item", RequestHandler(DeleteItemHandler))
	http.HandleFunc("/api/mov/item", RequestHandler(MoveItemHandler))

	http.HandleFunc("/", RequestHandler(func(res http.ResponseWriter, req *http.Request) {
		base_path := path.Join(__dir, "web/public/")

		req.URL.Path = strings.Replace(req.URL.Path, "../", "", -1)
		req.URL.Path = strings.Replace(req.URL.Path, "..\\", "", -1)

		if strings.HasSuffix(req.URL.Path, "/") {
			req.URL.Path += "index.html"
		}

		res.Header().Set("Cache-Control", "max-age=86400")

		if _, err := os.Stat(base_path + req.URL.Path); os.IsNotExist(err) {
			res.Header().Set("Content-Type", "text/html")

			content, err := ioutil.ReadFile(path.Join(base_path, "index.html"))
			if err != nil {
				log.Println(err.Error())
				return
			}

			res.Write(content)
		} else {
			ctype := ""

			if strings.HasSuffix(req.URL.Path, ".svg") {
				ctype = "image/svg+xml"
			} else if strings.HasSuffix(req.URL.Path, ".js") {
				ctype = "application/javascript"
			} else if strings.HasSuffix(req.URL.Path, ".css") {
				ctype = "text/css"
			} else {
				buff := make([]byte, 512)
				f, err := os.Open(base_path + req.URL.Path)
				if err != nil {
					log.Println(err.Error())
					return
				}

				defer f.Close()

				_, err = f.Read(buff)
				if err != nil {
					log.Println(err.Error())
					return
				}

				ctype = http.DetectContentType(buff)
			}

			res.Header().Set("Content-Type", ctype)

			content, err := ioutil.ReadFile(base_path + req.URL.Path)
			if err != nil {
				log.Println(err.Error())
				return
			}

			res.Write(content)
		}
	}))

	http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil)
}
