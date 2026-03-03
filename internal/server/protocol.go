package server

import "encoding/json"

// Message types
const (
	MsgJoin  = "join"  // client -> server
	MsgInput = "input" // client -> server
	MsgLeave = "leave" // client -> server (cancel queue)
	MsgLobby = "lobby" // server -> client
	MsgStart = "start" // server -> client
	MsgState = "state" // server -> client
	MsgFrame = "frame" // server -> client (lightweight position update)
	MsgOver  = "over"  // server -> client
)

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type JoinData struct {
	Name string `json:"name"`
	Mode string `json:"mode"` // "ai" or "pvp"
}

type InputData struct {
	Direction int `json:"direction"` // -1, 0, 1
}

type LobbyData struct {
	Waiting int `json:"waiting"`
}

type StartData struct {
	Side     string `json:"side"`     // "left" or "right"
	Opponent string `json:"opponent"` // opponent name
}

type OverData struct {
	Winner string `json:"winner"` // "left" or "right"
}

// FrameData is a lightweight position-only update sent between full state messages.
type FrameData struct {
	BX  float64 `json:"bx"`
	BY  float64 `json:"by"`
	BVX float64 `json:"bvx"`
	BVY float64 `json:"bvy"`
	LP  float64 `json:"lp"`
	RP  float64 `json:"rp"`
}

// typedEnvelope is used only for serialization — one json.Marshal call instead of two.
type typedEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func MakeEnvelope(typ string, data any) ([]byte, error) {
	return json.Marshal(typedEnvelope{Type: typ, Data: data})
}
