package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"
)

type mockServer struct {
	wg             *sync.WaitGroup
	path           string
	healthSequence []int
}

func newMockServer(path string, sequence ...int) mockServer {
	wg := sync.WaitGroup{}
	wg.Add(1)
	return mockServer{wg: &wg, path: path, healthSequence: sequence}
}

func (m *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != m.path {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if len(m.healthSequence) == 0 {
		w.WriteHeader(http.StatusOK)
		m.wg.Done()
		return
	}

	status := m.healthSequence[0]
	m.healthSequence = m.healthSequence[1:]
	w.WriteHeader(status)
}

func TestConstructHealthRemote(t *testing.T) {
	tests := map[string]struct {
		remote   *url.URL
		path     string
		tls      bool
		expected *url.URL
	}{
		"http": {
			remote: &url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
			},
			path: "/health",
			tls:  false,
			expected: &url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
				Path:   "/health",
			},
		},
		"https": {
			remote: &url.URL{
				Scheme: "https",
				Host:   "localhost:8080",
			},
			path: "/",
			tls:  true,
			expected: &url.URL{
				Scheme: "https",
				Host:   "localhost:8080",
				Path:   "/",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := constructHealthRemote(test.remote, test.path, test.tls)
			if actual.String() != test.expected.String() {
				t.Errorf("expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestHealthLaunchBadMethod(t *testing.T) {
	remote := url.URL{}

	health := NewHealth(HealthInfo{
		Path:     "/health",
		Method:   "*?", // Invalid method causes NewRequest to fail.
		TLS:      false,
		Interval: time.Millisecond,
	})

	go health.Launch(&remote)

	up := <-health.C()
	if up {
		t.Errorf("expected false, got %v", up)
	}
}

func TestHealthLaunchBadRequest(t *testing.T) {
	remote := url.URL{}

	health := NewHealth(HealthInfo{
		Path:     "/health",
		TLS:      false,
		Interval: time.Millisecond,
	})

	go health.Launch(&remote)

	up := <-health.C()
	if up {
		t.Errorf("expected false, got %v", up)
	}
}

func TestHealthLaunch(t *testing.T) {
	tests := map[string]struct {
		sequence []int
		expected []bool
	}{
		"success": {
			sequence: []int{http.StatusOK, http.StatusAccepted, http.StatusCreated},
			expected: []bool{true, true, true},
		},
		"failure": {
			sequence: []int{http.StatusNotFound, http.StatusNotFound, http.StatusNotFound},
			expected: []bool{false, false, false},
		},
		"alternate": {
			sequence: []int{http.StatusConflict, http.StatusOK, http.StatusNotFound, http.StatusAccepted, http.StatusCreated},
			expected: []bool{false, true, false, true, true},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := newMockServer("/health", test.sequence...)
			server := httptest.NewServer(&m)
			defer server.Close()

			remote, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("could not parse url %v", err)
			}

			health := NewHealth(HealthInfo{
				Path:     "/health",
				TLS:      false,
				Interval: time.Millisecond,
			})

			go health.Launch(remote)

			results := make([]bool, 0)

			for {
				up := <-health.C()
				results = append(results, up)
				if len(results) == len(test.expected) {
					break
				}
			}

			m.wg.Wait()

			if !slices.Equal(results, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, results)
			}
		})
	}
}
