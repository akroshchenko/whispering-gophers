// Skeleton to part 7 of the Whispering Gophers code lab.
//
// This program extends part 6 by adding a Peers type.
// The rest of the code is left as-is, so functionally there is no change.
//
// However we have added a peers_test.go file, so that running
//   go test
// from the package directory will test your implementation of the Peers type.
//
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"

	"github.com/campoy/whispering-gophers/util"
)

var (
	peerAddr = flag.String("peer", "", "ip:port to connect to")
	self     string
	messages = make(chan Message)
)

type Message struct {
	Addr, Body string
}

type Peers struct {
	ch map[string]chan<- Message
	mu sync.RWMutex
}

var peers = &Peers{
	ch: make(map[string]chan<- Message),
}

func (p *Peers) Add(addr string) <-chan Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.ch[addr]; ok {
		return nil
	}
	ch := make(chan Message)
	p.ch[addr] = ch
	return ch
}

func (p *Peers) Remove(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.ch, addr)
}

func (p *Peers) List() []chan<- Message {
	p.mu.RLock()
	defer p.mu.RUnlock()
	l := make([]chan<- Message, 0, len(p.ch))
	for _, ch := range p.ch {
		l = append(l, ch)
	}
	return l
}

func main() {
	flag.Parse()

	go dial(*peerAddr)
	go readInput()

	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}
	self = l.Addr().String()
	log.Println("Listening on ", self)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConnection(conn)
	}
}

func readInput() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		broadcast(Message{self, s.Text()})
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
}

func broadcast(m Message) {
	log.Println("Sending to all message", m)
	log.Println("Current peers %#v", peers.ch)
	for _, ch := range peers.List() {
		select {
		case ch <- m:
		default:
		}
	}
}

func handleConnection(c net.Conn) {
	defer c.Close()
	d := json.NewDecoder(c)
	for {
		var m Message
		err := d.Decode(&m)
		if err != nil {
			log.Fatal(err)
		}
		go dial(m.Addr)
		log.Printf("%#v", m)
	}
}

func dial(addr string) {
	if addr == self {
		return
	}

	ch := peers.Add(addr)
	if ch == nil {
		return
	}
	defer peers.Remove(addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	e := json.NewEncoder(conn)
	for m := range ch {
		err := e.Encode(m)
		if err != nil {
			log.Println(err)
			return
		}
	}

}
