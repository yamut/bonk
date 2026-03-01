package ssh

import (
	"bonk/internal/game"
	"fmt"
	"strings"
)

// Render converts the game state to an ASCII representation sized for the terminal.
func Render(state *game.GameState, width, height int) string {
	if width < 20 {
		width = 20
	}
	if height < 10 {
		height = 10
	}

	// Reserve lines for score header
	fieldH := height - 3
	fieldW := width

	// Build the field
	grid := make([][]rune, fieldH)
	for y := range grid {
		grid[y] = make([]rune, fieldW)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	// Scale from logical (800x600) to terminal
	scaleX := float64(fieldW) / float64(game.FieldWidth)
	scaleY := float64(fieldH) / float64(game.FieldHeight)

	// Center line
	cx := fieldW / 2
	for y := 0; y < fieldH; y++ {
		if y%2 == 0 {
			if cx >= 0 && cx < fieldW {
				grid[y][cx] = ':'
			}
		}
	}

	// Left paddle
	drawPaddle(grid, 1, state.LeftPaddle.Y, scaleY, fieldH)

	// Right paddle
	drawPaddle(grid, fieldW-2, state.RightPaddle.Y, scaleY, fieldH)

	// Ball
	bx := int(state.Ball.X * scaleX)
	by := int(state.Ball.Y * scaleY)
	if bx >= 0 && bx < fieldW && by >= 0 && by < fieldH {
		grid[by][bx] = 'o'
	}

	// Assemble output
	var sb strings.Builder

	// Score line
	scoreStr := fmt.Sprintf("%d   %d", state.LeftScore, state.RightScore)
	pad := (fieldW - len(scoreStr)) / 2
	if pad < 0 {
		pad = 0
	}
	sb.WriteString(strings.Repeat(" ", pad) + scoreStr + "\n")
	sb.WriteString(strings.Repeat("─", fieldW) + "\n")

	for _, row := range grid {
		sb.WriteString(string(row))
		sb.WriteString("\n")
	}

	return sb.String()
}

func drawPaddle(grid [][]rune, x int, centerY float64, scaleY float64, fieldH int) {
	cy := int(centerY * scaleY)
	paddleH := int(float64(game.PaddleHeight) * scaleY)
	if paddleH < 2 {
		paddleH = 2
	}
	top := cy - paddleH/2
	bottom := cy + paddleH/2

	for y := top; y <= bottom; y++ {
		if y >= 0 && y < fieldH && x >= 0 && x < len(grid[0]) {
			grid[y][x] = '|'
		}
	}
}

func RenderLobby(waiting int, width, height int) string {
	var sb strings.Builder

	title := []string{
		"╔══╗ ╔═══╗╔╗ ╔╗╔╗ ╔╗",
		"║╔╗║ ║╔═╗║║╚═╝║║║╔╝║",
		"║╚╝╚╗║║ ║║║╔╗ ║║╚╝╔╝",
		"║╔═╗║║║ ║║║║╚╗║║╔╗║ ",
		"║╚═╝║║╚═╝║║║ ║║║║║╚╗",
		"╚═══╝╚═══╝╚╝ ╚╝╚╝╚═╝",
	}

	// Vertical centering
	totalLines := len(title) + 6
	topPad := (height - totalLines) / 2
	if topPad < 0 {
		topPad = 0
	}
	for i := 0; i < topPad; i++ {
		sb.WriteString("\n")
	}

	// Title
	for _, line := range title {
		pad := (width - len([]rune(line))) / 2
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(strings.Repeat(" ", pad) + line + "\n")
	}

	sb.WriteString("\n")

	// Waiting count
	info := fmt.Sprintf("%d player(s) waiting", waiting)
	pad := (width - len(info)) / 2
	if pad < 0 {
		pad = 0
	}
	sb.WriteString(strings.Repeat(" ", pad) + info + "\n\n")

	// Menu
	options := []string{
		"[1] Play vs AI",
		"[2] Play vs Human",
		"[q] Quit",
	}
	for _, opt := range options {
		pad := (width - len(opt)) / 2
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(strings.Repeat(" ", pad) + opt + "\n")
	}

	return sb.String()
}

func RenderWaiting(waiting int, width, height int) string {
	var sb strings.Builder

	lines := []string{
		"Waiting for opponent...",
		"",
		fmt.Sprintf("%d player(s) in queue", waiting),
		"",
		"[esc] Cancel",
	}

	topPad := (height - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}
	for i := 0; i < topPad; i++ {
		sb.WriteString("\n")
	}
	for _, line := range lines {
		pad := (width - len(line)) / 2
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(strings.Repeat(" ", pad) + line + "\n")
	}

	return sb.String()
}

func RenderGameOver(winner string, mySide string, width, height int) string {
	var sb strings.Builder

	var msg string
	if winner == mySide {
		msg = "YOU WIN!"
	} else {
		msg = "YOU LOSE"
	}

	topPad := height / 2
	for i := 0; i < topPad-1; i++ {
		sb.WriteString("\n")
	}

	pad := (width - len(msg)) / 2
	if pad < 0 {
		pad = 0
	}
	sb.WriteString(strings.Repeat(" ", pad) + msg + "\n\n")

	again := "[r] Play again  [q] Quit"
	pad = (width - len(again)) / 2
	if pad < 0 {
		pad = 0
	}
	sb.WriteString(strings.Repeat(" ", pad) + again + "\n")

	return sb.String()
}
