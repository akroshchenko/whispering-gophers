// Solution to part 3 of the Whispering Gophers code lab.
//
// This program listens on the host and port specified by the -listen flag.
// For each incoming connection, it launches a goroutine that reads and decodes
// JSON-encoded messages from the connection and prints them to standard
// output.
//
// You can test this program by running it in one terminal:
// 	$ part3 -listen=localhost:8000
// And running part2 in another terminal:
// 	$ part2 -dial=localhost:8000
// Lines typed in the second terminal should appear as JSON objects in the
// first terminal.
//
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
)

type Message struct {
	Body string
}

var listenAddr = flag.String("listen", "", "Local address to server the connection on")

func main() {
	flag.Parse()

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", ln.Addr())

	for {
		c, err := ln.Accept()
		if err != nil {
			log.Println("Problems with serving connections from", err)
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	log.Println("serving new connection")
	// defer c.Close()
	d := json.NewDecoder(c)
	var m Message

	for {
		err := d.Decode(&m)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("%#v\n", m)
	}

}
