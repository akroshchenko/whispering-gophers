package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"part10-client/util"
)

var (
	peerAddr     = flag.String("peer", "", "peer host:port")
	numOfClients = flag.Int("num", 3, "number of parallel clients to spawn")
	self         string
)

type Message struct {
	ID   string
	Addr string
	Body string
}

func RunClient(addr string, ml int) {
	ip, err := util.ExternalIP()
	if err != nil {
		log.Fatalf("could not find active non-loopback address: %v", err)
	}

	l, err := net.Listen("tcp4", ip+":0")
	if err != nil {
		log.Fatal(err)
	}
	ip = l.Addr().String() // ip with port
	l.Close()

	log.Println("Starting new client on ", ip)

	go func() {
		// Run fake listing. This listiner will not handle somwhow any connections - just allow to connect to itself.
		ls, err := net.Listen("tcp4", ip)
		if err != nil {
			log.Fatal(err)
		}
		defer ls.Close()

		for {
			c, err := ls.Accept()
			if err != nil {
				log.Fatal(err)
			}
			defer c.Close()
		}
	}()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	e := json.NewEncoder(c)

	for {
		m := Message{
			ID:   util.RandomID(),
			Addr: ip,
			Body: util.RandomString(ml, "abcdefghijklmnorstuvxyz"+"ABCDEFGHIJKLMNORSTUVXYZ"),
		}
		err := e.Encode(m)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(3 * time.Second)
	}

}

func main() {
	flag.Parse()

	for i := 0; i < *numOfClients; i++ {
		go RunClient(*peerAddr, 30)
	}
	fmt.Scanln()
}
