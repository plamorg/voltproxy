// Package health provides utilitis to check the health of a service at a regular interval.
package health

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"
)

const (
	defaultHealthPath     = "/"
	defaultHealthInterval = 30 * time.Second
	defaultHealthTimeout  = 5 * time.Second
	defaultHealthMethod   = http.MethodGet
)

// Info describes a service Health capability.
type Info struct {
	Path     string        `yaml:"path"`
	TLS      bool          `yaml:"tls"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Method   string        `yaml:"method"`
}

// Result is the result of a health check.
type Result struct {
	Up       bool
	Err      error
	Endpoint string
}

// LogValue returns a slog.Value for the result, ensuring that the error is displayed properly.
func (h Result) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("up", h.Up),
		slog.Any("error", h.Err),
		slog.String("endpoint", h.Endpoint))
}

// Checker is the interface that wraps the basic methods for a health checker.
type Checker interface {
	Launch(remoteFunc func(w http.ResponseWriter, r *http.Request) (*url.URL, error))
	Up() bool
	Check() <-chan Result
}

// Health periodically checks the health of a service.
type Health struct {
	Info
	http.Handler

	c        chan Result
	resMutex sync.RWMutex
	res      Result
}

// New creates a new Health.
func New(info Info) *Health {
	if info.Path == "" {
		info.Path = defaultHealthPath
	}
	if info.Interval == 0 {
		info.Interval = defaultHealthInterval
	}
	if info.Timeout == 0 {
		info.Timeout = defaultHealthTimeout
	}
	if info.Method == "" {
		info.Method = defaultHealthMethod
	}
	return &Health{
		Info:     info,
		c:        make(chan Result),
		resMutex: sync.RWMutex{},
		res:      Result{},
	}
}

// Launch starts the periodic health check.
// A remoteFunc is used to get the service's remote URL in the case that the remote URL is dynamic.
// This remote is then used to construct the health remote URL that will be used for the health check.
func (h *Health) Launch(remoteFunc func(w http.ResponseWriter, r *http.Request) (*url.URL, error)) {
	ticker := time.NewTicker(h.Interval)
	for {
		w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
		remote, err := remoteFunc(w, r)
		if err != nil {
			h.resMutex.Lock()
			h.res = Result{Up: false, Endpoint: "", Err: err}
			h.resMutex.Unlock()
			h.c <- h.res
			continue
		}

		healthRemote := constructHealthRemote(remote, h.Path, h.TLS)
		status, err := h.requestStatus(healthRemote)
		up := status >= http.StatusOK && status < http.StatusBadRequest
		h.resMutex.Lock()
		h.res = Result{Up: up, Endpoint: healthRemote.String(), Err: err}
		h.resMutex.Unlock()
		h.c <- h.res

		<-ticker.C
	}
}

// Up returns whether the service is up.
func (h *Health) Up() bool {
	h.resMutex.RLock()
	defer h.resMutex.RUnlock()
	return h.res.Up
}

// Check returns a channel that will receive the health result on each check.
func (h *Health) Check() <-chan Result {
	return h.c
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

func (h *Health) requestStatus(healthRemote *url.URL) (int, error) {
	req, err := http.NewRequest(h.Method, healthRemote.String(), nil)
	if err != nil {
		return 0, err
	}

	client := &http.Client{
		Timeout: h.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
