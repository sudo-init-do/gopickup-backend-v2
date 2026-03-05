package notification

import (
	"context"
	"encoding/json"
	"gopickup/internal/db"
	"log"
	"sync"
)

type PubSubMessage struct {
	Room    string `json:"room"`
	Message []byte `json:"message"`
}

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

	// Metrics
	Metrics *Metrics
}

type Metrics struct {
	ActiveConnections int64
	MessagesSent      int64
	MessagesReceived  int64
	Errors            int64
	mu                sync.RWMutex
}

func (m *Metrics) GetStats() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]int64{
		"active_connections": m.ActiveConnections,
		"messages_sent":      m.MessagesSent,
		"messages_received":  m.MessagesReceived,
		"errors":             m.Errors,
	}
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Rooms:      make(map[string]map[*Client]bool),
		Metrics:    &Metrics{},
	}
}

func (h *Hub) Run() {
	go h.runMetrics()

	// Start Redis subscriber if available
	if redisClient := db.GetRedis(); redisClient != nil {
		go h.subscribeRedis()
	}

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.Clients == nil {
				h.Clients = make(map[*Client]bool)
			}
			h.Clients[client] = true
			h.Metrics.IncrementConnections()
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				h.removeClient(client)
				h.Metrics.DecrementConnections()
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			// Broadcast to all connected clients (system wide)
			// In Redis mode, we might want to publish this too, but usually Broadcast is for specific rooms
			// or global announcements.
			// For now, keep local broadcast logic.
			h.mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
					h.Metrics.IncrementMessagesSent()
				default:
					close(client.Send)
					delete(h.Clients, client)
					h.Metrics.DecrementConnections()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) runMetrics() {
	// Simple logger for now, could be pushed to Prometheus/Graphite
	for {
		// Log metrics every minute? No, just keep them in memory for the endpoint.
		// We could reset counters here if we wanted rate/minute.
		// For now, cumulative counters are fine.
		return
	}
}

// Metrics methods
func (m *Metrics) IncrementConnections() {
	m.mu.Lock()
	m.ActiveConnections++
	m.mu.Unlock()
}

func (m *Metrics) DecrementConnections() {
	m.mu.Lock()
	m.ActiveConnections--
	m.mu.Unlock()
}

func (m *Metrics) IncrementMessagesSent() {
	m.mu.Lock()
	m.MessagesSent++
	m.mu.Unlock()
}

func (m *Metrics) IncrementMessagesReceived() {
	m.mu.Lock()
	m.MessagesReceived++
	m.mu.Unlock()
}

func (m *Metrics) IncrementErrors() {
	m.mu.Lock()
	m.Errors++
	m.mu.Unlock()
}

func (h *Hub) subscribeRedis() {
	redisClient := db.GetRedis()
	pubsub := redisClient.Subscribe(context.Background(), "gopickup:ws:broadcast")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var data PubSubMessage
		if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
			log.Printf("Failed to unmarshal redis message: %v", err)
			continue
		}
		// Dispatch to local room
		h.broadcastToLocalRoom(data.Room, data.Message)
	}
}

// broadcastToLocalRoom sends to locally connected clients only
func (h *Hub) broadcastToLocalRoom(room string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.Rooms[room]; ok {
		for client := range clients {
			select {
			case client.Send <- message:
			default:
				// Skip if blocked
			}
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
	// If Redis is enabled, publish to Redis so other instances get it
	if redisClient := db.GetRedis(); redisClient != nil {
		data := PubSubMessage{
			Room:    room,
			Message: message,
		}
		payload, err := json.Marshal(data)
		if err == nil {
			redisClient.Publish(context.Background(), "gopickup:ws:broadcast", payload)
			// Return here? No, we still want to broadcast locally to our own clients.
			// However, if we subscribe to the same channel, we will receive it back and broadcast it twice.
			// To avoid this, we can rely SOLELY on the Redis subscription loop to trigger the local broadcast.
			// Yes, that is the standard pattern.
			return
		} else {
			log.Printf("Failed to marshal redis message: %v", err)
		}
	}

	// Fallback or Local-only mode
	h.broadcastToLocalRoom(room, message)
}
