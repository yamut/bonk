package main

import (
	"bonk/internal/config"
	"bonk/internal/server"
	sshrv "bonk/internal/ssh"
	"bonk/internal/web"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Load()

	hub := server.NewHub()
	hub.SetAIFactory(server.NewAIPlayer)
	go hub.BroadcastLobby()

	// HTTP + WebSocket server
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: web.Handler(hub),
	}
	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// SSH server
	sshServer, err := sshrv.NewServer(cfg, hub)
	if err != nil {
		log.Fatalf("SSH server init error: %v", err)
	}
	go sshrv.ListenAndServe(sshServer)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	sshrv.Shutdown(sshServer)
	httpServer.Close()
}
