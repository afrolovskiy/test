// +build amd64

package main

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Websocket settings
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	// Metrics
	reqCount uint64 // Total number of processed requests
	wsCount  int64  // Total number of websocket clients
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	log.Printf("%s ws: connected", r.RemoteAddr)
	defer func() { log.Printf("%s ws: disconnected", r.RemoteAddr) }()

	atomic.AddInt64(&wsCount, 1)
	defer func() { atomic.AddInt64(&wsCount, -1) }()

	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
					log.Printf("%s ws: failed to send ping: %s", r.RemoteAddr, err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Printf("%s ws: failed to read message: %s", r.RemoteAddr, err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}
	}
}

func sleepHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(&reqCount, 1)
	log.Printf("%s sleep: handled request", r.RemoteAddr)
	time.Sleep(50 * time.Millisecond)
	w.WriteHeader(http.StatusOK)
}

func metrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var oldReqCount uint64
	for {
		select {
		case <-ticker.C:
			curReqCount := atomic.LoadUint64(&reqCount)
			log.Printf("metrics: rps=%d ws=%d", curReqCount-oldReqCount, atomic.LoadInt64(&wsCount))
			oldReqCount = curReqCount
		}
	}
}

func main() {
	log.Printf("starting test-server")
	go metrics()

	http.HandleFunc("/sleep", sleepHandler)
	http.HandleFunc("/ws", wsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
