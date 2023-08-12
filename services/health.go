package services

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// HealthInfo describes a service Health capability.
type HealthInfo struct {
	Path     string        `yaml:"path"`
	TLS      bool          `yaml:"tls"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Method   string        `yaml:"method" default:"GET"`
}

// Health periodically checks the health of a service.
type Health struct {
	HealthInfo

	c  chan bool
	up bool
}

// NewHealth creates a new Health.
func NewHealth(info HealthInfo) *Health {
	return &Health{
		HealthInfo: info,
		c:          make(chan bool),
		up:         true,
	}
}

// Up returns the current health status.
func (h *Health) Up() bool {
	return h.up
}

func constructHealthRemote(remote *url.URL, path string, tls bool) *url.URL {
	healthRemote := *remote
	healthRemote.Path = path
	if tls {
		healthRemote.Scheme = "https"
	} else {
		healthRemote.Scheme = "http"
	}
	return &healthRemote
}

// Launch starts the periodic health check.
func (h *Health) Launch(serviceRemote *url.URL) {
	remote := constructHealthRemote(serviceRemote, h.Path, h.TLS)
	logger := slog.Default().With(
		slog.String("serviceRemote", remote.String()),
		slog.String("remote", remote.String()))

	ticker := time.NewTicker(h.Interval)
	for {
		<-ticker.C
		up, err := h.check(remote)
		if err != nil {
			logger.Warn("Health check failed", slog.Any("error", err))
			h.up = false
		} else {
			logger.Debug("Health check", slog.Bool("up", up), slog.Any("error", err))
			h.up = up
		}
		h.c <- h.up
	}
}

func (h *Health) check(remote *url.URL) (bool, error) {
	req, err := http.NewRequest(h.Method, remote.String(), nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{
		Timeout: h.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest, nil
}
