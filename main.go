package main

import (
	"log"

	"github.com/hekmon/transmissionrpc"
)

func main() {
	tc, err := transmissionrpc.New("localhost", "rpcuser", "rpcpass", nil)
	if err != nil {
		log.Fatalln("error intiializing tranmission client: ", err)
	}
	torrents, err := tc.TorrentGetAll()
	if err != nil {
		log.Fatalln("error getting torrents: ", err)
	}
	log.Printf("success getting %d torrents\n", len(torrents))
}
