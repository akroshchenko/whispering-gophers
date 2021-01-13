// Solution to part 2 of the Whispering Gophers code lab.
//
// This program extends part 1.
//
// It makes a connection the host and port specified by the -dial flag, reads
// lines from standard input and writes JSON-encoded messages to the network
// connection.
//
// You can test this program by installing and running the dump program:
// 	$ go get github.com/campoy/whispering-gophers/util/dump
// 	$ dump -listen=localhost:8000
// And in another terminal session, run this program:
// 	$ part2 -dial=localhost:8000
// Lines typed in the second terminal should appear as JSON objects in the
// first terminal.
//
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"time"
)

type Message struct {
	Body string
}

var (
	addr = flag.String("addr", "", "Address to connect to in format ip:port")
)

func main() {

	flag.Parse()

	c, err := net.Dial("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	if tc, ok := c.(*net.TCPConn); ok {
		err = tc.SetKeepAlive(true)
		if err != nil {
			log.Println("Error setting KeepAlive", err)
		}
		err = tc.SetKeepAlivePeriod(time.Second * 3)
		if err != nil {
			log.Println("Error setting SetKeepAlivePeriod", err)
		}

	}
	// log.Printf("Checking UDP conn: %#v", c.(*net.UDPConn))

	s := bufio.NewScanner(os.Stdin)
	e := json.NewEncoder(c)

	for s.Scan() {
		err := e.Encode(Message{Body: s.Text()})
		if err != nil {
			log.Fatal("Error while encoding", err)
		}
	}
	if err := s.Err(); err != nil {
		log.Fatal("Error reading the input", err)
	}
}
