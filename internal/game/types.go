package game

const (
	FieldWidth  = 800
	FieldHeight = 600

	PaddleWidth  = 10
	PaddleHeight = 80
	PaddleOffset = 20 // distance from wall

	BallSize         = 10
	BallInitialSpeed = 300.0 // pixels per second
	BallMaxSpeed     = 700.0
	BallSpeedInc     = 1.05 // 5% increase per hit

	WinScore = 11
	TickRate = 60
	TickDT   = 1.0 / float64(TickRate)

	MaxBounceAngle = 1.0 // radians (~57 degrees)
)

type Ball struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
}

type Paddle struct {
	Y float64 `json:"y"` // center Y position
}

type GameState struct {
	Ball        Ball   `json:"ball"`
	LeftPaddle  Paddle `json:"left_paddle"`
	RightPaddle Paddle `json:"right_paddle"`
	LeftScore   int    `json:"left_score"`
	RightScore  int    `json:"right_score"`
	Over        bool   `json:"over"`
	Winner      string `json:"winner"` // "left" or "right"
}
