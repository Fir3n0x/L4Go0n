package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

// File to handle websocket : communication between back and front

var (
	Clients   = make(map[*websocket.Conn]bool)
	ClientsMu = sync.Mutex{}
)

// Send a message to the front-end
func Broadcast(message string) {
	ClientsMu.Lock()
	defer ClientsMu.Unlock()
	for conn := range Clients {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			conn.Close()
			delete(Clients, conn)
		}
	}
}
