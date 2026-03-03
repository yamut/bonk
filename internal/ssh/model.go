package ssh

import (
	"bonk/internal/game"
	"bonk/internal/server"
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenLobby screen = iota
	screenWaiting
	screenGame
	screenOver
)

type serverMsg server.Envelope
type tickMsg struct{}

// stopPaddleMsg is sent after a keypress delay to stop the paddle.
// seq must match the model's current moveSeq or it's stale and ignored.
type stopPaddleMsg struct{ seq int }

type Model struct {
	hub     *server.Hub
	player  *server.Player
	screen  screen
	width   int
	height  int
	waiting int
	state   *game.GameState
	mySide  string
	winner  string
	dir     int // current input direction
	moveSeq int // incremented on each keypress; used to cancel stale stops
}

func NewModel(hub *server.Hub, width, height int) *Model {
	return &Model{
		hub:    hub,
		width:  width,
		height: height,
		screen: screenLobby,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, m.tick()

	case serverMsg:
		return m.handleServerMsg(server.Envelope(msg))

	case stopPaddleMsg:
		if msg.seq == m.moveSeq && m.dir != 0 {
			m.dir = 0
			m.sendDir()
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleServerMsg(env server.Envelope) (tea.Model, tea.Cmd) {
	switch env.Type {
	case server.MsgState:
		var state game.GameState
		if err := json.Unmarshal(env.Data, &state); err == nil {
			m.state = &state
		}
	case server.MsgFrame:
		var f server.FrameData
		if err := json.Unmarshal(env.Data, &f); err == nil && m.state != nil {
			m.state.Ball = game.Ball{X: f.BX, Y: f.BY, VX: f.BVX, VY: f.BVY}
			m.state.LeftPaddle.Y = f.LP
			m.state.RightPaddle.Y = f.RP
		}
	case server.MsgStart:
		var data server.StartData
		if err := json.Unmarshal(env.Data, &data); err == nil {
			m.mySide = data.Side
			m.screen = screenGame
		}
	case server.MsgLobby:
		var data server.LobbyData
		if err := json.Unmarshal(env.Data, &data); err == nil {
			m.waiting = data.Waiting
		}
	case server.MsgOver:
		var data server.OverData
		if err := json.Unmarshal(env.Data, &data); err == nil {
			m.winner = data.Winner
			m.screen = screenOver
		}
		return m, nil // stop ticking on game over
	}
	// Keep the tick loop alive
	return m, m.tick()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenLobby:
		switch msg.String() {
		case "1":
			return m, m.joinGame("ai")
		case "2":
			return m, m.joinGame("pvp")
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case screenWaiting:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			// Cancel queue
			if m.player != nil {
				m.hub.Leave(m.player)
				m.hub.RemoveClient(m.player)
			}
			m.screen = screenLobby
			m.player = nil
			return m, nil
		}

	case screenGame:
		switch msg.String() {
		case "up", "w":
			m.dir = -1
			m.moveSeq++
			m.sendDir()
			return m, m.scheduleStop()
		case "down", "s":
			m.dir = 1
			m.moveSeq++
			m.sendDir()
			return m, m.scheduleStop()
		case "q", "ctrl+c":
			if m.player != nil {
				m.player.Close()
			}
			return m, tea.Quit
		}

	case screenOver:
		switch msg.String() {
		case "r":
			if m.player != nil {
				m.hub.RemoveClient(m.player)
			}
			m.screen = screenLobby
			m.state = nil
			m.player = nil
			m.mySide = ""
			m.winner = ""
			m.dir = 0
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// scheduleStop returns a Cmd that sends a stopPaddleMsg after a short delay.
// If another keypress arrives before the delay, moveSeq will have incremented
// and this stop will be ignored as stale.
func (m *Model) scheduleStop() tea.Cmd {
	seq := m.moveSeq
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return stopPaddleMsg{seq: seq}
	})
}

func (m *Model) joinGame(mode string) tea.Cmd {
	m.player = server.NewPlayer("SSHPlayer")
	m.hub.AddClient(m.player)
	m.hub.Join(m.player, mode)

	if mode == "pvp" {
		m.screen = screenWaiting
	}

	// Start the tick loop that drains server messages
	return m.tick()
}

// tick drains messages from the player's Send channel and schedules next tick.
func (m *Model) tick() tea.Cmd {
	return func() tea.Msg {
		if m.player == nil {
			time.Sleep(time.Second / 30)
			return tickMsg{}
		}

		// Wait a short interval then drain messages
		time.Sleep(time.Second / 60)

		// Drain all pending messages, return the most important one
		var latest *server.Envelope
		for {
			select {
			case msg, ok := <-m.player.Send:
				if !ok {
					if latest != nil {
						return serverMsg(*latest)
					}
					return tickMsg{}
				}
				var env server.Envelope
				if err := json.Unmarshal(msg, &env); err == nil {
					// Start/over messages take priority — return immediately
					if env.Type == server.MsgStart || env.Type == server.MsgOver {
						return serverMsg(env)
					}
					latest = &env
				}
			default:
				if latest != nil {
					return serverMsg(*latest)
				}
				return tickMsg{}
			}
		}
	}
}

func (m *Model) sendDir() {
	if m.player == nil {
		return
	}
	data, _ := json.Marshal(server.InputData{Direction: m.dir})
	env := server.Envelope{Type: server.MsgInput, Data: data}
	select {
	case m.player.Recv <- env:
	default:
	}
}

func (m *Model) View() string {
	switch m.screen {
	case screenLobby:
		return RenderLobby(m.waiting, m.width, m.height)
	case screenWaiting:
		return RenderWaiting(m.waiting, m.width, m.height)
	case screenGame:
		if m.state == nil {
			return "Waiting for game state...\n"
		}
		return Render(m.state, m.width, m.height)
	case screenOver:
		return RenderGameOver(m.winner, m.mySide, m.width, m.height)
	}
	return ""
}
