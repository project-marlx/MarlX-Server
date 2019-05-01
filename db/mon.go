// Package db provides frequently used
// functionality for working with MongoDB
// databases.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo"
)

// StoredFile describes a file stored
// in the database, and keeps relevant
// information about it.
type StoredFile struct {
	Name         string    `bson:"name"`
	UniqueId     string    `bson:"uniqueId"`
	Size         int64     `bson:"size"`
	ActualSize   int64     `bson:"actualSize`
	MIMEType     string    `bson:"MIMEType"`
	Salt         string    `bson:"salt"`
	CTokens      [][]byte  `bson:"cTokens"`
	ParentDir    string    `bson:"parentDir"`
	IsDir        bool      `bson:"isDir"`
	DirContent   [][]byte  `bson:"dirContent"`
	CreationTime time.Time `bson:"creationTime"`
}

// Returns string representation of a
// StoredFile instance.
func (sf *StoredFile) String() string {
	return fmt.Sprintf("StoredFile{Name: %s, ClientTokens: [...], IsDir: %t, DirContent: [...]}",
		sf.Name, sf.IsDir)
}

// StoredClient represents a client stored
// in the MongoDB database.
type StoredClient struct {
	Token      []byte `bson:"token"`
	Hostname   string `bson:"hostname"`
	FreeBytes  int64  `bson:"freeBytes"`
	TotalBytes int64  `bson:"totalBytes"`
	MTU        int64  `bson:"MTU"`
}

// Returns string representation of a StoredClient
// instance.
func (sc *StoredClient) String() string {
	return fmt.Sprintf("StoredClient{Token: %x, FreeBytes: %d, TotalBytes: %d}",
		sc.Token, sc.FreeBytes, sc.TotalBytes)
}

// StoredUser describes a user stored
// in the MongoDB database.
type StoredUser struct {
	Token    []byte   `bson:"token"`
	Username string   `bson:"username"`
	Email    string   `bson:"email"`
	Password string   `bson:"password"`
	Clients  [][]byte `bson:"clients"`
	// Files    map[string]StoredFile `bson:"files"`
}

// Returns string representation of a
// StoredUser instance.
func (su *StoredUser) String() string {
	return fmt.Sprintf("StoredUser{Token: %x, Username: %s, Password: %s, Clients: [...], Files: map[]{...}}",
		su.Token, su.Username, su.Password)
}

// MongoUser contains information of a
// MongoDB-User. It will be used when making
// a connection to a MongoDB.
type MongoUser struct {
	Username string
	Password string
}

// ConnectDB is an alias for:
// ConnectDBOnPort(hostname, 27017, dbname, user)
func ConnectDB(hostname string, dbname string, user MongoUser) (context.Context, *mongo.Client, error) {
	return ConnectDBOnPort(hostname, 27017, dbname, user)
}

// ConnectDBOnPort connects to a MongoDB database
// on the given host + port and returns the
// MongoDB-Client + an error (if one occured)
func ConnectDBOnPort(hostname string, port uint16, dbname string, user MongoUser) (context.Context, *mongo.Client, error) {
	fmt.Println("[" + time.Now().String() + "]: Connecting to MongoDB ... ")

	ctx := context.Background()
	client, err := mongo.Connect(ctx, fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", user.Username, user.Password, hostname, port, dbname))

	return ctx, client, err
}

// ConnectDBNoUser is an alias for:
// ConnectDBNoUserOnPort(hostname, 27107, dbname)
func ConnectDBNoUser(hostname string, dbname string) (context.Context, *mongo.Client, error) {
	return ConnectDBNoUserOnPort(hostname, 27017, dbname)
}

// ConnectDBNoUserOnPort connects to a MongoDB
// database on the give host + port without authentications.
// It returns a MongoDB-Client + an error (if any occured)
func ConnectDBNoUserOnPort(hostname string, port uint16, dbname string) (context.Context, *mongo.Client, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, fmt.Sprintf("mongodb://%s:%d/%s", hostname, port, dbname))

	return ctx, client, err
}
