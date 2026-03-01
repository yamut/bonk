package server

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Hub struct {
	mu       sync.Mutex
	queue    []*Player // PvP waiting queue
	sessions map[string]*GameSession
	clients  map[*Player]bool   // all connected clients (for lobby broadcast)
	createAI func(*Hub) *Player // factory for AI players
}

func NewHub() *Hub {
	return &Hub{
		sessions: make(map[string]*GameSession),
		clients:  make(map[*Player]bool),
	}
}

func (h *Hub) SetAIFactory(f func(*Hub) *Player) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.createAI = f
}

func (h *Hub) AddClient(p *Player) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[p] = true
}

func (h *Hub) RemoveClient(p *Player) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, p)
	// Remove from queue if waiting
	for i, qp := range h.queue {
		if qp == p {
			h.queue = append(h.queue[:i], h.queue[i+1:]...)
			break
		}
	}
}

func (h *Hub) Join(p *Player, mode string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Prevent duplicate joins — ignore if already queued
	for _, qp := range h.queue {
		if qp == p {
			return
		}
	}

	if mode == "ai" {
		if h.createAI == nil {
			log.Println("hub: AI factory not set")
			return
		}
		aiPlayer := h.createAI(h)
		sessionID := uuid.New().String()
		session := NewGameSession(sessionID, p, aiPlayer, h)
		h.sessions[sessionID] = session
		go session.Run()
		return
	}

	// PvP mode
	h.queue = append(h.queue, p)
	if len(h.queue) >= 2 {
		left := h.queue[0]
		right := h.queue[1]
		h.queue = h.queue[2:]

		sessionID := uuid.New().String()
		session := NewGameSession(sessionID, left, right, h)
		h.sessions[sessionID] = session
		go session.Run()
	}
}

// Leave removes a player from the PvP queue. Returns true if they were queued.
func (h *Hub) Leave(p *Player) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, qp := range h.queue {
		if qp == p {
			h.queue = append(h.queue[:i], h.queue[i+1:]...)
			return true
		}
	}
	return false
}

func (h *Hub) RemoveSession(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, id)
}

func (h *Hub) WaitingCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.queue)
}

// BroadcastLobby sends the waiting player count to all connected clients periodically.
func (h *Hub) BroadcastLobby() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		count := len(h.queue)
		clients := make([]*Player, 0, len(h.clients))
		for c := range h.clients {
			clients = append(clients, c)
		}
		h.mu.Unlock()

		msg, err := MakeEnvelope(MsgLobby, LobbyData{Waiting: count})
		if err != nil {
			continue
		}

		for _, c := range clients {
			select {
			case c.Send <- msg:
			case <-c.Done:
			default:
			}
		}
	}
}

func unmarshalData(data json.RawMessage, v any) error {
	return json.Unmarshal(data, v)
}
