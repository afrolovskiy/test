package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var rc chan struct{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	log.Printf("connected %s", r.RemoteAddr)

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Print("started to send ping")
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
					log.Printf("failed to send ping: %s", err)
					return
				}
				log.Print("succesfully sended ping")
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
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("failed to read message: %s", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			http.Error(w, "", http.StatusOK)
			return
		}
		log.Printf("dreceived message with type=%d: %s", messageType, message)
	}
}

func sleepHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(50 * time.Millisecond)
	log.Printf("handled request %s", r.RemoteAddr)
	rc <- struct{}{}
}

func aggregate() {
	var m sync.Mutex
	var count int

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.Lock()
			rps := count
			count = 0
			m.Unlock()
			log.Printf("rps=%d", rps/10.0)

		case <-rc:
			m.Lock()
			count++
			m.Unlock()
		}
	}
}

func main() {
	rc = make(chan struct{}, 10000)
	go aggregate()

	log.Printf("server started")
	http.HandleFunc("/sleep", sleepHandler)
	http.HandleFunc("/ws", wsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
