package server

import (
	"bonk/internal/game"
	"log"
	"time"
)

type GameSession struct {
	ID             string
	Engine         *game.Engine
	Left           *Player
	Right          *Player
	hub            *Hub
	lastLeftScore  int
	lastRightScore int
}

func NewGameSession(id string, left, right *Player, hub *Hub) *GameSession {
	return &GameSession{
		ID:     id,
		Engine: game.NewEngine(),
		Left:   left,
		Right:  right,
		hub:    hub,
	}
}

func (gs *GameSession) Run() {
	defer func() {
		gs.hub.RemoveSession(gs.ID)
	}()

	// Notify players of game start
	startLeft, _ := MakeEnvelope(MsgStart, StartData{Side: "left", Opponent: gs.Right.Name})
	startRight, _ := MakeEnvelope(MsgStart, StartData{Side: "right", Opponent: gs.Left.Name})
	gs.safeSend(gs.Left, startLeft)
	gs.safeSend(gs.Right, startRight)

	ticker := time.NewTicker(time.Second / game.TickRate)
	defer ticker.Stop()

	var leftDir, rightDir int

	for range ticker.C {
		// Drain inputs
		leftDir = gs.drainInput(gs.Left, leftDir)
		rightDir = gs.drainInput(gs.Right, rightDir)

		// Check for disconnects
		select {
		case <-gs.Left.Done:
			gs.Engine.State.Over = true
			gs.Engine.State.Winner = "right"
			gs.broadcastState()
			gs.broadcastOver()
			return
		case <-gs.Right.Done:
			gs.Engine.State.Over = true
			gs.Engine.State.Winner = "left"
			gs.broadcastState()
			gs.broadcastOver()
			return
		default:
		}

		gs.Engine.Tick(leftDir, rightDir)
		gs.broadcastUpdate()

		if gs.Engine.State.Over {
			gs.broadcastOver()
			return
		}
	}
}

func (gs *GameSession) drainInput(p *Player, current int) int {
	dir := current
	for {
		select {
		case env := <-p.Recv:
			if env.Type == MsgInput {
				var input InputData
				if err := unmarshalData(env.Data, &input); err == nil {
					dir = input.Direction
				}
			}
		default:
			return dir
		}
	}
}

func (gs *GameSession) broadcastUpdate() {
	s := &gs.Engine.State
	if s.LeftScore != gs.lastLeftScore || s.RightScore != gs.lastRightScore || s.Over {
		gs.lastLeftScore = s.LeftScore
		gs.lastRightScore = s.RightScore
		gs.broadcastState()
	} else {
		gs.broadcastFrame()
	}
}

func (gs *GameSession) broadcastState() {
	msg, err := MakeEnvelope(MsgState, gs.Engine.State)
	if err != nil {
		log.Printf("session %s: marshal state: %v", gs.ID, err)
		return
	}
	gs.safeSend(gs.Left, msg)
	gs.safeSend(gs.Right, msg)
}

func (gs *GameSession) broadcastFrame() {
	s := &gs.Engine.State
	msg, err := MakeEnvelope(MsgFrame, FrameData{
		BX:  s.Ball.X,
		BY:  s.Ball.Y,
		BVX: s.Ball.VX,
		BVY: s.Ball.VY,
		LP:  s.LeftPaddle.Y,
		RP:  s.RightPaddle.Y,
	})
	if err != nil {
		log.Printf("session %s: marshal frame: %v", gs.ID, err)
		return
	}
	gs.safeSend(gs.Left, msg)
	gs.safeSend(gs.Right, msg)
}

func (gs *GameSession) broadcastOver() {
	msg, _ := MakeEnvelope(MsgOver, OverData{Winner: gs.Engine.State.Winner})
	gs.safeSend(gs.Left, msg)
	gs.safeSend(gs.Right, msg)
}

func (gs *GameSession) safeSend(p *Player, msg []byte) {
	select {
	case p.Send <- msg:
	case <-p.Done:
	default:
		// Channel full, drop message
	}
}
