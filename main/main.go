package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/MattMoony/MarlX-Server/db"
	"github.com/MattMoony/MarlX-Server/socks"

	"github.com/MattMoony/MarlX-Server/crypto/RSAWrapper"
	client_lib "github.com/MattMoony/MarlX-Server/marlx/client"
	"github.com/MattMoony/MarlX-Server/marlx/clients"
	"github.com/MattMoony/MarlX-Server/web"
)

var connected_clients map[string]*client_lib.Client

var (
	streams       = map[string]socks.WebStream{}
	streams_mutex = sync.RWMutex{}
)

func main() {
	fmt.Println(" [MarlX-Server]: Booting up ... ")
	connected_clients = make(map[string]*client_lib.Client, 0)

	tcpListener, err := socks.GetTCPListener("127.0.0.1")
	if err != nil {
		log.Panic(err)
	}
	defer tcpListener.Close()

	fmt.Printf(" [MarlX-Server]: Listening on :%s\n", strings.Split(tcpListener.Addr().String(), ":")[1])

	priv, err := RSAWrapper.GenerateKey()
	if err != nil {
		log.Panic(err)
	}

	dbctx, dbclient, err := db.ConnectDB("localhost", "marlx", db.MongoUser{"marlx-server", "Ce7dehyJJDjyGTLwwerkGGBswHhGxpu9"})
	if err != nil {
		log.Panic(err)
	}

	fmt.Println(" [MarlX-Server]: Connected to MongoDB ... ")

	go web.Start(connected_clients, priv, dbctx, dbclient, streams, streams_mutex)
	fmt.Println(" [MarlX-Server]: Web server has started ... ")

	for {
		conn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}

		go clients.HandleClient(conn, priv, connected_clients, dbctx, dbclient, streams, streams_mutex)
	}
}
