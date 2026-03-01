package server

import (
	"sync"

	"github.com/google/uuid"
)

type Player struct {
	ID        string
	Name      string
	Send      chan []byte   // outbound messages
	Recv      chan Envelope // inbound messages
	Done      chan struct{} // closed on disconnect
	closeOnce sync.Once
}

func NewPlayer(name string) *Player {
	return &Player{
		ID:   uuid.New().String(),
		Name: name,
		Send: make(chan []byte, 64),
		Recv: make(chan Envelope, 64),
		Done: make(chan struct{}),
	}
}

// Close safely closes the Done channel exactly once.
func (p *Player) Close() {
	p.closeOnce.Do(func() {
		close(p.Done)
	})
}
