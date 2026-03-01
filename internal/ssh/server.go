package ssh

import (
	"bonk/internal/config"
	"bonk/internal/server"
	"context"
	"log"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/ratelimiter"
	"golang.org/x/time/rate"
)

func NewServer(cfg *config.Config, hub *server.Hub) (*ssh.Server, error) {
	srv, err := wish.NewServer(
		wish.WithAddress(cfg.SSHAddr),
		wish.WithHostKeyPath(cfg.SSHHostKeyPath),
		wish.WithIdleTimeout(cfg.SSHIdleTimeout),
		wish.WithMaxTimeout(cfg.SSHMaxTimeout),
		wish.WithMiddleware(
			bubbletea.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
				pty, _, _ := s.Pty()
				m := NewModel(hub, pty.Window.Width, pty.Window.Height)
				return m, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			activeterm.Middleware(),
			maxConnsMiddleware(cfg.SSHMaxConns),
			ratelimiter.Middleware(
				ratelimiter.NewRateLimiter(rate.Every(cfg.SSHRateInterval), cfg.SSHRateBurst, cfg.SSHRateCacheSize),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func ListenAndServe(srv *ssh.Server) {
	log.Printf("SSH server listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("SSH server error: %v", err)
	}
}

func Shutdown(srv *ssh.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func maxConnsMiddleware(max int64) wish.Middleware {
	var active atomic.Int64
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			if active.Add(1) > max {
				active.Add(-1)
				wish.Fatalln(sess, "server full, try again later")
				return
			}
			defer active.Add(-1)
			next(sess)
		}
	}
}
