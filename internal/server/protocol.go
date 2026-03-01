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

func MakeEnvelope(typ string, data any) ([]byte, error) {
	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{Type: typ, Data: d})
}
