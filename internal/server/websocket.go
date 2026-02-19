package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// upgrader configures the WebSocket upgrade parameters.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development.
	},
}

// Hub manages WebSocket connections and broadcasts messages to all
// connected clients. It is the central coordination point for live reload
// notifications.
type Hub struct {
	mu         sync.Mutex
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	done       chan struct{}
}

// NewHub creates a new Hub ready to manage WebSocket connections.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		done:       make(chan struct{}),
	}
}

// Run starts the hub event loop. It processes register, unregister, and
// broadcast events until Stop is called.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					delete(h.clients, conn)
					conn.Close()
				}
			}
			h.mu.Unlock()

		case <-h.done:
			h.mu.Lock()
			for conn := range h.clients {
				conn.Close()
				delete(h.clients, conn)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Stop shuts down the hub event loop and closes all client connections.
func (h *Hub) Stop() {
	close(h.done)
}

// Broadcast sends a message to all connected WebSocket clients.
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		// Drop message if broadcast channel is full.
	}
}

// HandleWS upgrades an HTTP connection to a WebSocket and registers it
// with the hub. The connection is automatically unregistered when the
// client disconnects.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	h.register <- conn

	// Read loop: wait for the client to disconnect.
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// ClientCount returns the current number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}
