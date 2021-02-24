// TODO: fix gracefull shutdow when several client have been connected

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/campoy/whispering-gophers/util"
	"github.com/pkg/profile"
)

var (
	self             string
	interceptSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

type Message struct {
	ID   string
	Addr string
	Body string
}

func main() {

	flag.Parse()

	if *cpuprofile != "" {
		defer profile.Start(profile.CPUProfile, profile.ProfilePath(*cpuprofile), profile.NoShutdownHook).Stop()
	}

	if *memprofile != "" {
		defer profile.Start(profile.MemProfile, profile.ProfilePath(*memprofile), profile.NoShutdownHook).Stop()
	}

	// Q: does it make sence to use buffered channel or not?
	// There was interesting problem https://github.com/golang/go/issues/38290
	// https://go.googlesource.com/proposal/+/master/design/24543-non-cooperative-preemption.md
	sigChan := make(chan os.Signal, len(interceptSignals))
	signal.Notify(sigChan, interceptSignals...)
	log.Println("The next OS signal will be intercepted by this program: ", interceptSignals)

	l, err := util.Listen()
	if err != nil {
		log.Println(err)
		return
	}

	// Q: is it ok to handle the return err from close like this?
	defer func() {
		log.Println("Defer: Closing listener")
		if err := l.Close(); err != nil {
			// TODO fix the error:
			// Error while closing listener: close tcp4 192.168.1.186:52508: use of closed network connection
			log.Printf("Defer: Error while closing listener: %v\n", err)
		}
	}()

	self = l.Addr().String()

	log.Println("Listening on", self)

	newConns := make(chan net.Conn)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Printf("Error with accepting connection: %v", err)
				newConns <- nil
				return
			}
			newConns <- c
		}
	}()

	quit := make(chan struct{})

	var wg sync.WaitGroup

	go func() {
		for {
			select {
			case <-quit:
				log.Printf("Receiving connections: Recieved quit signal")
				// l.Close()
				if err := l.Close(); err != nil {
					log.Printf("Receiving connections: Error while closing listener: %v\n", err)
				}
				return
			case con := <-newConns:
				if con == nil {
					return
				}
				wg.Add(1)
				go serve(quit, &wg, con)
			}
		}
	}()

	s := <-sigChan
	log.Printf("Recieved interaption signal: %v\n", s)

	// Close serving
	log.Println("Sending a quit signal to outgoing connections")
	close(quit)

	wg.Wait()

	// make sure no one writes to the closed channel

	// Close outgoing connections
	log.Println("Closing channels for outgoing connections")
	for _, c := range peers.List() {
		close(c)
	}
}

var peers = &Peers{m: make(map[string]chan<- Message)}

type Peers struct {
	m  map[string]chan<- Message
	mu sync.RWMutex
}

// Add creates and returns a new channel for the given peer address.
// If an address already exists in the registry, it returns nil.
func (p *Peers) Add(addr string) <-chan Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.m[addr]; ok {
		return nil
	}
	ch := make(chan Message)
	p.m[addr] = ch
	return ch
}

// Remove deletes the specified peer from the registry.
func (p *Peers) Remove(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.m, addr)
}

// List returns a slice of all active peer channels.
func (p *Peers) List() []chan<- Message {
	p.mu.RLock()
	defer p.mu.RUnlock()
	l := make([]chan<- Message, 0, len(p.m))
	for _, ch := range p.m {
		l = append(l, ch)
	}
	return l
}

func broadcast(quit <-chan struct{}, m Message) {
	for _, ch := range peers.List() {
		// Q: is it ok to do so? (how to add quit channel a higher priority?)
		select {
		case <-quit:
			log.Println("Brodcas: Received the quit signal")
			return
		default:
		}
		select {
		case ch <- m:
		default:
			log.Println("Broadcast: dropping the messages")
			// Okay to drop messages sometimes.
		}
	}
}

func serve(quit <-chan struct{}, wg *sync.WaitGroup, c net.Conn) {
	defer wg.Done()
	defer c.Close()
	d := json.NewDecoder(c)
	for {
		select {
		case <-quit:
			log.Println("Serving connection: Received the signal to end serving connection\n")
			return
		default:
		}

		var m Message
		err := d.Decode(&m)
		if err != nil {
			log.Printf("Serving connection: error with decoding message from %v: %w\n", c.RemoteAddr(), err)
			return
		}
		if Seen(m.ID) {
			continue
		}
		fmt.Printf("%#v\n", m)
		broadcast(quit, m)
		go dial(quit, m.Addr)
	}
}

// func readInput() {
// 	log.Println("Attaching Stdin to the server input...")
// 	s := bufio.NewScanner(os.Stdin)
// 	for s.Scan() {
// 		m := Message{
// 			ID:   util.RandomID(),
// 			Addr: self,
// 			Body: s.Text(),
// 		}
// 		Seen(m.ID)
// 		broadcast(m)
// 	}
// 	if err := s.Err(); err != nil {
// 		log.Fatal(err)
// 	}
// }

func dial(quit <-chan struct{}, addr string) {
	if addr == self {
		return // Don't try to dial self.
	}

	ch := peers.Add(addr)
	if ch == nil {
		return // Peer already connected.
	}
	defer peers.Remove(addr)

	log.Println("Dealing to ", addr)

	con, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(addr, err)
		return
	}
	defer con.Close()

	e := json.NewEncoder(con)
	for m := range ch {
		select {
		case <-quit:
			return
		default:
		}

		err := e.Encode(m)
		if err != nil {
			log.Printf("Dialing: Error to encode message from %v: %v\n", addr, err)
			return
		}
	}
}

var seenIDs = struct {
	m map[string]bool
	sync.Mutex
}{m: make(map[string]bool)}

// Seen returns true if the specified id has been seen before.
// If not, it returns false and marks the given id as "seen".
func Seen(id string) bool {
	seenIDs.Lock()
	ok := seenIDs.m[id]
	seenIDs.m[id] = true
	seenIDs.Unlock()
	return ok
}
