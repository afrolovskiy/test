// +build amd64

package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http server address")
var wsCount = flag.Int("ws", 1, "number of concurrent websocket connection")
var load = flag.Int("load", 1, "number of concurrent requests per second")

const writeWait = 10 * time.Second

// rc is channel which is used for rps calculating.
var rc chan struct{}

// sleep sends HTTP-request to /sleep endpoint.
func sleep(rid int) {
	defer func() { rc <- struct{}{} }()

	u := url.URL{Scheme: "http", Host: *addr, Path: "/sleep"}
	log.Printf("#%d: sending request to url=%s", rid, u.String())

	resp, err := http.Post(u.String(), "application/json", nil)
	if err != nil {
		log.Printf("#%d: failed to send request: %s", rid, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("#%d: response code=%d", rid, resp.StatusCode)
}

// ws sets up websocket connection with server, sends message to server once a second.
func ws(id int) {
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("#%d: connected to ws url=%s", id, u.String())

	// Connect to the server
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("#%d: failed to open websocket connection: %s", id, err)
		return
	}
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer conn.Close()
		defer close(done)

		// Print all message which were received from server
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("#%d: failed to read message: %s", id, err)
				return
			}
			log.Printf("#%d: received message with type=%d: %s", id, messageType, message)
		}
	}()

	// Send one message to server every second.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			err := conn.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Printf("failed to write messsage: %s", err)
				return
			}
		}
	}
}

// aggregate calculates RPS metric and prints it to the log.
func aggregate() {
	var m sync.Mutex
	var count int

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.Lock()
			rps := count
			count = 0
			m.Unlock()
			log.Printf("rps=%d", rps)

		case <-rc:
			m.Lock()
			count++
			m.Unlock()
		}
	}
}

func main() {
	flag.Parse()

	rc = make(chan struct{}, 10000)
	go aggregate()

	// Start concurrent websocket connections.
	for i := 0; i < *wsCount; i++ {
		go ws(i)
	}

	// Send batch of requests each second
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var rid int
	for {
		select {
		case <-ticker.C:
			for i := 0; i < *load; i++ {
				go sleep(rid)
				rid++
			}
		}
	}
}
