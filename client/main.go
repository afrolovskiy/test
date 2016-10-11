// +build amd64

package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http server address")
var wsRPS = flag.Int("ws", 1, "number of concurrent websocket connection")
var sleepRPS = flag.Int("sleep", 1, "number of concurrent requests per second")
var verbose = flag.Bool("verbose", false, "show more debug info")
var showMetrics = flag.Bool("show-metrics", false, "show metrics")
var sendWsMessage = flag.Bool("ws-test-message", false, "send test messages to websocket")

// Websocket settings
const writeWait = 10 * time.Second

var (
	// Metrics
	wsCount     int64  // Total number of websocket clients
	reqSucCount uint64 // Total number of success request
	reqErrCount uint64 // Total number of error requests
	// Semaphor to control the number of concurrent websockets
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

	if *verbose {
		log.Printf("sleep: sent request code=%d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusOK {
		atomic.AddUint64(&reqSucCount, 1)
	} else {
		atomic.AddUint64(&reqErrCount, 1)
	}
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

	if *verbose {
		log.Printf("ws: connected to the server from add=%s", conn.LocalAddr())
	}

	atomic.AddInt64(&wsCount, 1)
	defer func() { atomic.AddInt64(&wsCount, -1) }()

	if *sendWsMessage {
		go func() {
			defer conn.Close()
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
					if *verbose {
						log.Printf("ws: sent message")
					}
				}
			}
		}()
	}

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Printf("ws: failed to read message: %s", err)
			return
		}
		if *verbose {
			log.Printf("ws: received message")
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
			if *showMetrics {
				log.Printf("metrics: rps=%d errs=%d ws=%d",
					curReqSucCount-oldReqSucCount,
					curReqErrCount-oldReqErrCount,
					atomic.LoadInt64(&wsCount))
			}
			oldReqSucCount = curReqSucCount
			oldReqErrCount = curReqErrCount
		}
	}
}

func worker(id int) {
	for {
		// Send only 1 request per second:
		// <time before request> <request execution time> <time after request> = 1s

		// TODO: use crypto/rand
		before := rand.Float64()
		<-time.After(time.Duration(before*1000) * time.Millisecond)

		start := time.Now()
		sleep()
		elapsed := time.Since(start)

		after := 1 - elapsed.Seconds() - before
		<-time.After(time.Duration(int(after*1000)) * time.Millisecond)
	}
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().Unix())
	go metrics()

	wsSem = make(chan struct{}, *wsRPS)
	go func() {
		for {
			wsSem <- struct{}{}
			go listen()
		}
	}()

	for i := 0; i < *sleepRPS; i++ {
		go worker(i)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
