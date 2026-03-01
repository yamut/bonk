package game

import "math"

const paddleSpeed = 400.0 // pixels per second

type Engine struct {
	State GameState
}

func NewEngine() *Engine {
	e := &Engine{}
	e.Reset()
	return e
}

func (e *Engine) Reset() {
	e.State = GameState{
		Ball: Ball{
			X:  FieldWidth / 2,
			Y:  FieldHeight / 2,
			VX: BallInitialSpeed,
			VY: 0,
		},
		LeftPaddle:  Paddle{Y: FieldHeight / 2},
		RightPaddle: Paddle{Y: FieldHeight / 2},
	}
}

// Tick advances the game by one frame.
// leftDir/rightDir: -1 = up, 0 = still, 1 = down
func (e *Engine) Tick(leftDir, rightDir int) {
	if e.State.Over {
		return
	}

	s := &e.State

	// Move paddles
	s.LeftPaddle.Y += float64(leftDir) * paddleSpeed * TickDT
	s.RightPaddle.Y += float64(rightDir) * paddleSpeed * TickDT

	// Clamp paddles within field
	halfPaddle := float64(PaddleHeight) / 2
	s.LeftPaddle.Y = clamp(s.LeftPaddle.Y, halfPaddle, FieldHeight-halfPaddle)
	s.RightPaddle.Y = clamp(s.RightPaddle.Y, halfPaddle, FieldHeight-halfPaddle)

	// Move ball
	s.Ball.X += s.Ball.VX * TickDT
	s.Ball.Y += s.Ball.VY * TickDT

	// Wall bounce (top/bottom)
	halfBall := float64(BallSize) / 2
	if s.Ball.Y-halfBall <= 0 {
		s.Ball.Y = halfBall
		s.Ball.VY = -s.Ball.VY
	} else if s.Ball.Y+halfBall >= FieldHeight {
		s.Ball.Y = FieldHeight - halfBall
		s.Ball.VY = -s.Ball.VY
	}

	// Left paddle collision
	leftPaddleX := float64(PaddleOffset + PaddleWidth)
	if s.Ball.VX < 0 && s.Ball.X-halfBall <= leftPaddleX && s.Ball.X-halfBall >= float64(PaddleOffset) {
		if s.Ball.Y >= s.LeftPaddle.Y-halfPaddle && s.Ball.Y <= s.LeftPaddle.Y+halfPaddle {
			s.Ball.X = leftPaddleX + halfBall
			e.reflectBall(&s.LeftPaddle, 1)
		}
	}

	// Right paddle collision
	rightPaddleX := float64(FieldWidth - PaddleOffset - PaddleWidth)
	if s.Ball.VX > 0 && s.Ball.X+halfBall >= rightPaddleX && s.Ball.X+halfBall <= float64(FieldWidth-PaddleOffset) {
		if s.Ball.Y >= s.RightPaddle.Y-halfPaddle && s.Ball.Y <= s.RightPaddle.Y+halfPaddle {
			s.Ball.X = rightPaddleX - halfBall
			e.reflectBall(&s.RightPaddle, -1)
		}
	}

	// Scoring
	if s.Ball.X-halfBall <= 0 {
		s.RightScore++
		e.resetBall(-1)
	} else if s.Ball.X+halfBall >= FieldWidth {
		s.LeftScore++
		e.resetBall(1)
	}

	// Win check
	if s.LeftScore >= WinScore {
		s.Over = true
		s.Winner = "left"
	} else if s.RightScore >= WinScore {
		s.Over = true
		s.Winner = "right"
	}
}

// reflectBall bounces the ball off a paddle.
// dirX is 1 for left paddle (ball goes right) or -1 for right paddle (ball goes left).
func (e *Engine) reflectBall(paddle *Paddle, dirX float64) {
	s := &e.State

	// Offset from paddle center: -1 to 1
	offset := (s.Ball.Y - paddle.Y) / (float64(PaddleHeight) / 2)
	offset = clamp(offset, -1, 1)

	angle := offset * MaxBounceAngle
	speed := math.Sqrt(s.Ball.VX*s.Ball.VX+s.Ball.VY*s.Ball.VY) * BallSpeedInc
	if speed > BallMaxSpeed {
		speed = BallMaxSpeed
	}

	s.Ball.VX = dirX * speed * math.Cos(angle)
	s.Ball.VY = speed * math.Sin(angle)
}

func (e *Engine) resetBall(dirX float64) {
	e.State.Ball = Ball{
		X:  FieldWidth / 2,
		Y:  FieldHeight / 2,
		VX: dirX * BallInitialSpeed,
		VY: 0,
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
