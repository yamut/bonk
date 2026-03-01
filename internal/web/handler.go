package web

import (
	"bonk/internal/server"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Handler(hub *server.Hub) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWS(w, r, hub)
	})
	return mux
}

func handleWS(w http.ResponseWriter, r *http.Request, hub *server.Hub) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	player := server.NewPlayer("WebPlayer")
	hub.AddClient(player)

	// Write pump
	go func() {
		defer conn.Close()
		for {
			select {
			case msg, ok := <-player.Send:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-player.Done:
				return
			}
		}
	}()

	// Read pump
	go func() {
		defer func() {
			hub.RemoveClient(player)
			player.Close()
			conn.Close()
		}()

		joined := false
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var env server.Envelope
			if err := json.Unmarshal(msg, &env); err != nil {
				continue
			}
			switch env.Type {
			case server.MsgJoin:
				if joined {
					continue // ignore duplicate joins
				}
				var join server.JoinData
				if err := json.Unmarshal(env.Data, &join); err == nil {
					player.Name = join.Name
					hub.Join(player, join.Mode)
					joined = true
				}
			case server.MsgLeave:
				if hub.Leave(player) {
					joined = false
				}
			default:
				select {
				case player.Recv <- env:
				default:
				}
			}
		}
	}()
}
