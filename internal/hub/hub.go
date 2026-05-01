package hub

import (
	"encoding/json"
	"log"
	"sync"
)

type Client struct {
	ID   string
	Send chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
}

func New() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			total := len(h.clients)
			h.mu.Unlock()
			log.Printf("ws client connected: %s (total: %d)", client.ID, total)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			total := len(h.clients)
			h.mu.Unlock()
			log.Printf("ws client disconnected: %s (total: %d)", client.ID, total)

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Broadcast(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("hub marshal error: %v", err)
		return
	}
	h.broadcast <- data
}

func (h *Hub) BroadcastRaw(data []byte) {
	h.broadcast <- data
}

func (h *Hub) NumClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
