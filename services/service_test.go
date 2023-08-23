package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/plamorg/voltproxy/middlewares"
)

func TestHandlerSuccess(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r, w := httptest.NewRequest(http.MethodGet, "http://example.com", nil), httptest.NewRecorder()

	serverURL, err := url.Parse(okServer.URL)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	services := map[string]*Service{
		"foo.example.com": {
			TLS:    false,
			Router: NewRedirect(url.URL{}),
		},
		"example.com": {
			TLS:    false,
			Router: NewRedirect(*serverURL),
		},
	}

	expectedHost := strings.Split(okServer.URL, "://")[1]

	Handler(services).ServeHTTP(w, r)
	if r.Host != expectedHost {
		t.Errorf("expected host %s got host %s", expectedHost, r.Host)
	}
}

type badRouter struct{}

func (b badRouter) Route(_ http.ResponseWriter, _ *http.Request) (*url.URL, error) {
	return nil, fmt.Errorf("bad router")
}

func TestHandlerErrors(t *testing.T) {
	services := map[string]*Service{
		"foo.example.com": {},
		"bad.example.com": {
			Router: badRouter{},
		},
	}
	tests := map[string]struct {
		handler      http.Handler
		target       string
		expectedCode int
	}{
		"no service found": {
			handler:      Handler(services),
			target:       "example.com",
			expectedCode: http.StatusNotFound,
		},
		"bad router": {
			handler:      Handler(services),
			target:       "bad.example.com",
			expectedCode: http.StatusInternalServerError,
		},
		"no service found TLS": {
			handler:      TLSHandler(services),
			target:       "foo.example.com",
			expectedCode: http.StatusNotFound,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://"+test.target, nil)
			test.handler.ServeHTTP(w, r)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != test.expectedCode {
				t.Errorf("expected code %d got code %d", test.expectedCode, res.StatusCode)
			}
		})
	}
}

func TestHandlerRedirectToTLS(t *testing.T) {
	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	services := map[string]*Service{
		"example.com": {
			TLS:    true,
			Router: NewRedirect(url.URL{}),
		},
	}

	// Access a TLS service through HTTP and expect to get redirected to HTTPS.
	Handler(services).ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	location, err := res.Location()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedLocation := "https://example.comhttp://example.com"
	if location.String() != expectedLocation {
		t.Errorf("expected location %s got location %s", expectedLocation, location.String())
	}
}

type mockMiddleware struct{}

func (m *mockMiddleware) Handle(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Mock-Middleware", "true")
	})
}

func TestHandlerAddsMiddlewares(t *testing.T) {
	services := map[string]*Service{
		"example.com": {
			Middlewares: []middlewares.Middleware{
				&mockMiddleware{},
			},
			Router: NewRedirect(url.URL{}),
		},
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	Handler(services).ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.Header.Get("X-Mock-Middleware") != "true" {
		t.Errorf("expected middleware to be added")
	}
}
