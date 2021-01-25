package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"

	"github.com/campoy/whispering-gophers/util"
)

var (
	self             string
	interceptSignals = []os.Signal{syscall.SIGINT}
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
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		// defer pprof.StopCPUProfile()
		defer func() {
			// fmt.Println("Ending even with force quit")
			pprof.StopCPUProfile()
		}()
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}

	// Q: does it make sence to use buffered channel or not? (tried with buffered channel - does not work)
	// There was interesting problem https://github.com/golang/go/issues/38290
	// https://go.googlesource.com/proposal/+/master/design/24543-non-cooperative-preemption.md
	sigChan := make(chan os.Signal, len(interceptSignals))
	signal.Notify(sigChan, interceptSignals...)
	log.Println("The next OS signal will be intercepted by this program: ", interceptSignals)

	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}

	// Q: is it ok to handle the return err from close like this?
	defer func() {
		log.Println("Closing listener")
		if err := l.Close(); err != nil {
			log.Printf("Error while closing listener: %w\n", err)
		}
	}()

	self = l.Addr().String()

	log.Println("Listening on", self)

	var connCount int64
	quit := make(chan struct{})

	for {
		select {
		case s := <-sigChan:
			log.Printf("Recieved interaption signal: %v\n", s)

			// Close serving
			log.Println("Sending a quit signal to outgoing connections")
			for i := 0; int64(i) < connCount; i++ {
				quit <- struct{}{}
			}
			// Close outgoing connections
			log.Println("Closing channels for outgoing connections")
			for _, c := range peers.List() {
				close(c)
			}

			// Q: is it ok to close a listiner as soon as posible? (as a result we will call close several times)
			log.Println("Closing listener")
			if err := l.Close(); err != nil {
				log.Printf("Error while closing listener: %w\n", err)
			}
			break
		default:
			c, err := l.Accept()
			if err != nil {
				log.Printf("Error with accepting connection: %w", err)
				return
			}
			connCount++
			go serve(quit, c)
		}
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

func broadcast(m Message) {
	for _, ch := range peers.List() {
		select {
		case ch <- m:
		default:
			// Okay to drop messages sometimes.
		}
	}
}

func serve(quit <-chan struct{}, c net.Conn) {
	defer c.Close()
	d := json.NewDecoder(c)
	for {
		select {
		case <-quit:
			log.Println("Serving connection: Recieved the signal to end serving connection\n")
			break
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
		broadcast(m)
		go dial(m.Addr)
	}
}

func readInput() {
	log.Println("Attaching Stdin to the server input...")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		m := Message{
			ID:   util.RandomID(),
			Addr: self,
			Body: s.Text(),
		}
		Seen(m.ID)
		broadcast(m)
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
}

func dial(addr string) {
	if addr == self {
		return // Don't try to dial self.
	}

	ch := peers.Add(addr)
	if ch == nil {
		return // Peer already connected.
	}
	defer peers.Remove(addr)

	log.Println("Dealing to ", addr)

	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(addr, err)
		return
	}
	defer c.Close()

	e := json.NewEncoder(c)
	for m := range ch {
		err := e.Encode(m)
		if err != nil {
			log.Println(addr, err)
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
