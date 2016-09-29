// +build amd64

package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http server address")
var wsRPS = flag.Int("ws", 1, "number of concurrent websocket connection")
var sleepRPS = flag.Int("sleep", 1, "number of concurrent requests per second")

// Websocket settings
const writeWait = 10 * time.Second

var (
	// Metrics
	wsCount     int64  // Total number of websocket clients
	reqSucCount uint64 // Total number of success request
	reqErrCount uint64 // Total number of error requests

	wsSem chan struct{}
)

func sleep() {
	u := url.URL{Scheme: "http", Host: *addr, Path: "/sleep"}
	resp, err := http.Post(u.String(), "application/json", nil)
	if err != nil {
		log.Printf("sleep: failed to send request: %s", err)
		atomic.AddUint64(&reqErrCount, 1)
		return
	}

	defer resp.Body.Close()
	atomic.AddUint64(&reqSucCount, 1)
}

func listen() {
	defer func() { <-wsSem }()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("ws: failed to open websocket connection: %s", err)
		return
	}
	defer conn.Close()

	atomic.AddInt64(&wsCount, 1)
	defer func() { atomic.AddInt64(&wsCount, -1) }()

	done := make(chan struct{})
	go func() {
		defer conn.Close()
		defer close(done)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				log.Printf("ws: failed to read message: %s", err)
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			err := conn.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Printf("ws: failed to write messsage: %s", err)
				return
			}
		}
	}
}

func metrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var oldReqSucCount uint64
	var oldReqErrCount uint64
	for {
		select {
		case <-ticker.C:
			curReqSucCount := atomic.LoadUint64(&reqSucCount)
			curReqErrCount := atomic.LoadUint64(&reqErrCount)
			log.Printf("metrcis: rps=%d errs=%d ws=%d",
				curReqSucCount-oldReqSucCount,
				curReqErrCount-oldReqErrCount,
				atomic.LoadInt64(&wsCount))
			oldReqSucCount = curReqSucCount
			oldReqErrCount = curReqErrCount
		}
	}
}

func main() {
	flag.Parse()
	go metrics()

	wsSem = make(chan struct{}, *wsRPS)
	go func() {
		for {
			wsSem <- struct{}{}
			go listen()
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for i := 0; i < *sleepRPS; i++ {
				go sleep()
			}
		}
	}
}
