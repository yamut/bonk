package game

import (
	"math"
	"math/rand"
)

type AI struct {
	Difficulty     float64 // 0.1 to 0.95
	lastHumanScore int
	lastAIScore    int

	// Stable target — recalculated once when ball starts approaching
	targetY   float64
	hasTarget bool
}

func NewAI() *AI {
	return &AI{Difficulty: 0.5}
}

// Adapt adjusts difficulty based on score changes.
func (ai *AI) Adapt(humanScore, aiScore int) {
	changed := false
	if humanScore > ai.lastHumanScore {
		ai.Difficulty += 0.02
		changed = true
	}
	if aiScore > ai.lastAIScore {
		ai.Difficulty -= 0.04
		changed = true
	}
	ai.lastHumanScore = humanScore
	ai.lastAIScore = aiScore

	if ai.Difficulty < 0.1 {
		ai.Difficulty = 0.1
	}
	if ai.Difficulty > 0.95 {
		ai.Difficulty = 0.95
	}

	if changed {
		ai.hasTarget = false
	}
}

// Decide returns the paddle direction for the AI (-1, 0, or 1).
func (ai *AI) Decide(state *GameState) int {
	ball := state.Ball
	paddleY := state.RightPaddle.Y

	ballMovingToward := ball.VX > 0

	// When ball is moving away, just stay put — looks natural
	if !ballMovingToward {
		ai.hasTarget = false
		return 0
	}

	// Compute target once when ball starts coming toward AI
	if !ai.hasTarget {
		ai.recalcTarget(state)
		ai.hasTarget = true
	}

	// Dead zone — don't micro-adjust if close enough to target
	deadZone := float64(PaddleHeight) * (0.4 + (1-ai.Difficulty)*0.3)
	diff := ai.targetY - paddleY
	if math.Abs(diff) < deadZone {
		return 0
	}
	if diff < 0 {
		return -1
	}
	return 1
}

// recalcTarget computes a target Y with intentional error baked in.
func (ai *AI) recalcTarget(state *GameState) {
	predicted := ai.predictBallY(state)

	// Error: at low difficulty, the AI's target can be off
	// Error range scales from ±135px at difficulty 0.1 to ±7px at difficulty 0.95
	maxError := (1 - ai.Difficulty) * 150
	offset := (rand.Float64()*2 - 1) * maxError

	ai.targetY = clamp(predicted+offset, float64(PaddleHeight)/2, FieldHeight-float64(PaddleHeight)/2)
}

// predictBallY estimates where the ball will cross the AI's paddle x-line.
func (ai *AI) predictBallY(state *GameState) float64 {
	ball := state.Ball

	if ball.VX <= 0 {
		return ball.Y
	}

	targetX := float64(FieldWidth - PaddleOffset - PaddleWidth)
	x := ball.X
	y := ball.Y
	vy := ball.VY
	vx := ball.VX

	halfBall := float64(BallSize) / 2

	dt := (targetX - x) / vx
	y += vy * dt

	// Simulate wall bounces
	for i := 0; i < 20; i++ {
		if y-halfBall >= 0 && y+halfBall <= FieldHeight {
			break
		}
		if y-halfBall < 0 {
			y = halfBall - (y - halfBall)
		}
		if y+halfBall > FieldHeight {
			y = (FieldHeight - halfBall) - (y + halfBall - FieldHeight)
		}
	}

	return y
}
