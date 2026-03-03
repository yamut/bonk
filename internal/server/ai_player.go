package server

import (
	"bonk/internal/game"
	"encoding/json"
	"time"
)

// NewAIPlayer creates a Player backed by an AI that reads state from Send
// and feeds input decisions into Recv.
func NewAIPlayer(hub *Hub) *Player {
	p := NewPlayer("AI")
	ai := game.NewAI()

	go func() {
		// 10Hz decision rate — produces smooth movement instead of 60Hz jitter.
		// The session reuses the last direction between decisions.
		ticker := time.NewTicker(time.Second / 10)
		defer ticker.Stop()

		var lastLeftScore, lastRightScore int

		for {
			select {
			case <-p.Done:
				return
			case <-ticker.C:
			}

			// Drain Send channel. Process all "state" messages for score
			// updates, but use the latest message for the AI decision.
			var lastMsg []byte
		drain:
			for {
				select {
				case msg := <-p.Send:
					// Peek at state messages to capture score updates
					var env Envelope
					if json.Unmarshal(msg, &env) == nil && env.Type == MsgState {
						var s game.GameState
						if json.Unmarshal(env.Data, &s) == nil {
							lastLeftScore = s.LeftScore
							lastRightScore = s.RightScore
						}
					}
					lastMsg = msg
				default:
					break drain
				}
			}

			if lastMsg == nil {
				continue
			}

			var env Envelope
			if err := json.Unmarshal(lastMsg, &env); err != nil {
				continue
			}

			var state game.GameState
			switch env.Type {
			case MsgState:
				if err := json.Unmarshal(env.Data, &state); err != nil {
					continue
				}
				lastLeftScore = state.LeftScore
				lastRightScore = state.RightScore
			case MsgFrame:
				var f FrameData
				if err := json.Unmarshal(env.Data, &f); err != nil {
					continue
				}
				state = game.GameState{
					Ball:        game.Ball{X: f.BX, Y: f.BY, VX: f.BVX, VY: f.BVY},
					LeftPaddle:  game.Paddle{Y: f.LP},
					RightPaddle: game.Paddle{Y: f.RP},
					LeftScore:   lastLeftScore,
					RightScore:  lastRightScore,
				}
			default:
				continue
			}

			ai.Adapt(state.LeftScore, state.RightScore)
			dir := ai.Decide(&state)

			inputEnv := Envelope{Type: MsgInput}
			inputEnv.Data, _ = json.Marshal(InputData{Direction: dir})

			select {
			case p.Recv <- inputEnv:
			default:
			}
		}
	}()

	return p
}
