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

	"part10-client/util"
)

var (
	peerAddr     = flag.String("peer", "", "peer host:port")
	numOfClients = flag.Int("num", 3, "number of parallel clients to spawn")
	self         string
)

// Q: Does it make sence to put the common code for client and server to a different package.
// Q: What are the principles of package/dir structure
type Message struct {
	ID   string
	Addr string
	Body string
}

func RunClient(sigChan <-chan os.Signal, wg *sync.WaitGroup, i int, addr string, ml int) {

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

	// Q: is it the right place to handle the closign?
	// TODO: think about handling error from Close()
	defer l.Close()

	ip = l.Addr().String()

	log.Printf("Starting listiner for Client # %v on %v\n", i, ip)

	go func() {
		for {
			select {
			case <-sigChan:
				break
			default:
			}

			c, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(ioutil.Discard, c)
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
		case <-sigChan:
			break
		default:
		}

		m := Message{
			ID:   util.RandomID(),
			Addr: ip,
			// TOOD: fix it. It will be needed for memory profiling (maybe)
			// Body: util.RandomString(ml, "abcdefghijklmnorstuvxyz"+"ABCDEFGHIJKLMNORSTUVXYZ"),
			Body: util.RandomID(),
		}
		err := e.Encode(m)
		if err != nil {
			log.Print(err)
			return
		}
		//time.Sleep(3 * time.Second)
	}

	// wg.Done()
}

func main() {
	flag.Parse()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan)

	var wg sync.WaitGroup

	wg.Add(*numOfClients)
	for i := 0; i < *numOfClients; i++ {
		go RunClient(sigChan, &wg, i, *peerAddr, 20)
	}
	wg.Wait()
}
