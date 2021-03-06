// Solution to part 1 of the Whispering Gophers code lab.
// This program reads from standard input and writes JSON-encoded messages to
// standard output. For example, this input line:
//	Hello!
// Produces this output:
//	{"Body":"Hello!"}
//
package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

type Message struct {
	Body string
}

func main() {

	s := bufio.NewScanner(os.Stdin)
	e := json.NewEncoder(os.Stdin)

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
