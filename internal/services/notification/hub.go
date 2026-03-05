package notification

import (
	"log"
	"sync"
)

type Client struct {
	Hub *Hub

	// The websocket connection.
	Send chan []byte

	UserID string
	Rooms  map[string]bool
	mu     sync.Mutex
}

type Hub struct {
	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	// Rooms map: roomName -> map[Client]bool
	Rooms map[string]map[*Client]bool

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Rooms:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				h.removeClient(client)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					// We can't delete here safely while iterating with RLock,
					// but usually we just skip or mark for deletion.
					// Simplification: just skip for now.
				}
			}
			h.mu.RUnlock()
		}
	}
}

// removeClient must be called with h.mu.Lock() held
func (h *Hub) removeClient(client *Client) {
	delete(h.Clients, client)
	close(client.Send)

	client.mu.Lock()
	defer client.mu.Unlock()

	for roomName := range client.Rooms {
		if clients, ok := h.Rooms[roomName]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.Rooms, roomName)
			}
		}
	}
	// Clear client rooms
	client.Rooms = make(map[string]bool)
}

func (h *Hub) Subscribe(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Rooms[room] == nil {
		h.Rooms[room] = make(map[*Client]bool)
	}
	h.Rooms[room][client] = true

	client.mu.Lock()
	if client.Rooms == nil {
		client.Rooms = make(map[string]bool)
	}
	client.Rooms[room] = true
	client.mu.Unlock()
}

func (h *Hub) Unsubscribe(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.Rooms[room]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.Rooms, room)
		}
	}

	client.mu.Lock()
	if client.Rooms != nil {
		delete(client.Rooms, room)
	}
	client.mu.Unlock()
}

func (h *Hub) BroadcastToRoom(room string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.Rooms[room]; ok {
		log.Printf("Broadcasting to room %s: %d clients", room, len(clients))
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				log.Printf("Client %s blocked/full", client.UserID)
				// Skip if blocked
			}
		}
	} else {
		log.Printf("Room %s not found or empty", room)
	}
}
