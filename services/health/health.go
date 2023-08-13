// Package health provides utilitis to check the health of a service at a regular interval.
package health

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"
)

// Info describes a service Health capability.
type Info struct {
	Path     string        `yaml:"path"`
	TLS      bool          `yaml:"tls"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Method   string        `yaml:"method" default:"GET"`
}

// Result is the result of a health check.
type Result struct {
	Status   int
	Err      error
	Endpoint string
}

// LogValue returns a slog.Value for the result, ensuring that the error is displayed properly.
func (h Result) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("status", h.Status),
		slog.Any("error", h.Err),
		slog.String("endpoint", h.Endpoint))
}

// Health periodically checks the health of a service.
type Health struct {
	Info

	c   chan Result
	res Result
}

// New creates a new Health.
func New(info Info) *Health {
	return &Health{
		Info: info,
		c:    make(chan Result),
		res:  Result{Endpoint: "", Status: 0, Err: nil},
	}
}

// Check returns a channel that will receive the health result on each check.
func (h *Health) Check() <-chan Result {
	return h.c
}

// Up returns the current health status.
func (h *Health) Up() bool {
	return h.res.Status >= http.StatusOK && h.res.Status < http.StatusBadRequest
}

// Launch starts the periodic health check.
// A remoteFunc is used to get the service's remote URL in the case that the remote URL is dynamic.
// This remote is then used to construct the health remote URL that will be used for the health check.
func (h *Health) Launch(remoteFunc func(w http.ResponseWriter, r *http.Request) (*url.URL, error)) {
	ticker := time.NewTicker(h.Interval)
	for {
		<-ticker.C
		w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
		remote, err := remoteFunc(w, r)
		if err != nil {
			h.res = Result{Endpoint: "", Status: 0, Err: err}
			h.c <- h.res
			continue
		}

		healthRemote := constructHealthRemote(remote, h.Path, h.TLS)
		status, err := h.requestStatus(healthRemote)
		h.res = Result{Endpoint: healthRemote.String(), Status: status, Err: err}
		h.c <- h.res
	}
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
