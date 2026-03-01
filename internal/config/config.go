package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr         string
	SSHAddr          string
	SSHHostKeyPath   string
	SSHIdleTimeout   time.Duration
	SSHMaxTimeout    time.Duration
	SSHMaxConns      int64
	SSHRateInterval  time.Duration
	SSHRateBurst     int
	SSHRateCacheSize int
}

func Load() *Config {
	var cfg Config

	flag.StringVar(&cfg.HTTPAddr, "http-addr", envOrDefault("HTTP_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.SSHAddr, "ssh-addr", envOrDefault("SSH_ADDR", ":2222"), "SSH listen address")
	flag.StringVar(&cfg.SSHHostKeyPath, "ssh-host-key", envOrDefault("SSH_HOST_KEY", ".ssh_host_key"), "path to SSH host key")

	idleTimeout := flag.String("ssh-idle-timeout", envOrDefault("SSH_IDLE_TIMEOUT", "5m"), "SSH idle timeout")
	maxTimeout := flag.String("ssh-max-timeout", envOrDefault("SSH_MAX_TIMEOUT", "30m"), "SSH max timeout")
	maxConns := flag.String("ssh-max-conns", envOrDefault("SSH_MAX_CONNS", "100"), "max concurrent SSH connections")
	rateInterval := flag.String("ssh-rate-interval", envOrDefault("SSH_RATE_INTERVAL", "1s"), "SSH rate limit interval")
	rateBurst := flag.String("ssh-rate-burst", envOrDefault("SSH_RATE_BURST", "3"), "SSH rate limit burst")
	rateCacheSize := flag.String("ssh-rate-cache", envOrDefault("SSH_RATE_CACHE", "256"), "SSH rate limiter cache size")

	flag.Parse()

	cfg.SSHIdleTimeout = parseDuration(*idleTimeout, 5*time.Minute)
	cfg.SSHMaxTimeout = parseDuration(*maxTimeout, 30*time.Minute)
	cfg.SSHMaxConns = parseInt64(*maxConns, 100)
	cfg.SSHRateInterval = parseDuration(*rateInterval, time.Second)
	cfg.SSHRateBurst = parseInt(*rateBurst, 3)
	cfg.SSHRateCacheSize = parseInt(*rateCacheSize, 256)

	return &cfg
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func parseInt64(s string, fallback int64) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func parseInt(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
