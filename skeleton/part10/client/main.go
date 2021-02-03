// TODO: fix gracefull shutdow when several client have been connected to remote server
package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"part10-client/util"
)

var (
	peerAddr         = flag.String("peer", "", "peer host:port")
	numOfClients     = flag.Int("num", 3, "number of parallel clients to spawn")
	self             string
	interceptSignals = []os.Signal{syscall.SIGINT}
	GlobalTimeout    = 10 * time.Minute
)

// Q: Does it make sence to put the common code for client and server to a different package.
// Q: What are the principles of package/dir structure
type Message struct {
	ID   string
	Addr string
	Body string
}

func RunClient(quit <-chan struct{}, wg *sync.WaitGroup, i int, addr string, ml int) {
	defer wg.Done()

	// Cannot make ip var global as ExternalIP returns 2 parameters
	ip, err := util.ExternalIP()
	if err != nil {
		log.Fatalf("could not find active non-loopback address: %v", err)
	}

	l, err := net.Listen("tcp4", ip+":0")
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		// log.Println("Closing listener")
		if err := l.Close(); err != nil {
			log.Printf("Error while closing listener: %v\n", err)
		}
	}()

	ip = l.Addr().String()

	log.Printf("Starting listiner for Client # %v on %v\n", i, ip)

	newConns := make(chan net.Conn)

	go func() {
		for {
			con, err := l.Accept()
			if err != nil {
				log.Printf("Error with accepting connection: %v", err)
				newConns <- nil
				return
			}
			newConns <- con
		}
	}()

	go func() {
		for {
			select {
			case <-quit:
				return
			case con := <-newConns:
				if con == nil {
					return
				}
				// Allow connections to dummy client listiner but do nothing
				io.Copy(ioutil.Discard, con)
			}
		}
	}()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Connecting from Client # %v (%v) to %v\n", i, c.LocalAddr().String(), addr)

	e := json.NewEncoder(c)

	for {
		select {
		case <-quit:
			break
		default:
			m := Message{
				ID:   util.RandomID(),
				Addr: ip,
				// TOOD: fix it. It will be needed for memory profiling (maybe)
				// Body: util.RandomString(ml, "abcdefghijklmnorstuvxyz"+"ABCDEFGHIJKLMNORSTUVXYZ"),
				Body: util.RandomID(),
			}
			err := e.Encode(m)
			if err != nil {
				log.Println(err)
				return
			}
			// time.Sleep(3 * time.Second)
		}
	}
}

func main() {
	flag.Parse()

	sigChan := make(chan os.Signal, len(interceptSignals))
	signal.Notify(sigChan, interceptSignals...)
	log.Println("The next OS signal will be intercepted by this program: ", interceptSignals)

	// Q: what is better: buffered or not? I am for buffered but needs to be checked.
	quit := make(chan struct{})

	var wg sync.WaitGroup

	wg.Add(*numOfClients)

	for i := 0; i < *numOfClients; i++ {
		go RunClient(quit, &wg, i, *peerAddr, 20)
	}

	select {
	case s := <-sigChan:
		log.Printf("Received inraption signal: %v\n", s)
	case <-time.After(time.Minute * 2):
		log.Println("Time out")
	}

	log.Println("Sending a quit signal to all running clients")
	close(quit)

	wg.Wait()
}
