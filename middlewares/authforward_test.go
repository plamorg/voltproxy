package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var teapotHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
})

func TestAuthForwardHandle(t *testing.T) {
	authServer := func(status int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
	}
	tests := map[string]struct {
		authServer   *httptest.Server
		expectedCode int
	}{
		"auth failure": {
			authServer:   authServer(http.StatusUnauthorized),
			expectedCode: http.StatusUnauthorized,
		},
		"auth success": {
			authServer:   authServer(http.StatusOK),
			expectedCode: http.StatusTeapot,
		},
		"auth success accepted": {
			authServer:   authServer(http.StatusAccepted),
			expectedCode: http.StatusTeapot,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a := NewAuthForward(test.authServer.URL, nil, nil)
			handler := a.Handle(teapotHandler)

			w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(w, r)

			if w.Code != test.expectedCode {
				t.Errorf("expected %d, got %d", test.expectedCode, w.Code)
			}
		})
	}
}

func TestAuthForwardHandleRequestHeaders(t *testing.T) {
	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Test-Header") != "test" {
			t.Errorf("test header was not forwarded")
		}
		if r.Header.Get("Not-Forwarded") != "" {
			t.Errorf("header was unexpectedly forwarded")
		}

		w.WriteHeader(http.StatusOK)
	}))

	a := NewAuthForward(verifyingAuthServer.URL, []string{"test-header"}, nil)

	handler := a.Handle(teapotHandler)

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Test-Header", "test")
	r.Header.Set("Not-Forwarded", "do not forward")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
	}
}

func TestAuthForwardHandleEmptyRequestHeaders(t *testing.T) {
	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no request headers are specified, we expect all headers to be forwarded.
		if r.Header.Get("Test-Header") != "test" {
			t.Errorf("test header was not forwarded")
		}
		w.WriteHeader(http.StatusOK)
	}))

	a := NewAuthForward(verifyingAuthServer.URL, nil, nil)

	handler := a.Handle(teapotHandler)

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Test-Header", "test")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
	}
}

func TestAuthForwardHandleResponseHeaders(t *testing.T) {
	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Test-Header", "test")
		w.Header().Set("Do-Not-Forward", "do not forward")
		w.WriteHeader(http.StatusOK)
	}))

	a := NewAuthForward(verifyingAuthServer.URL, nil, []string{"test-header"})

	handler := a.Handle(teapotHandler)

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if r.Header.Get("Test-Header") != "test" {
		t.Errorf("test header was not forwarded")
	}

	if r.Header.Get("Do-Not-Forward") != "" {
		t.Errorf("header was unexpectedly forwarded")
	}

	if w.Code != http.StatusTeapot {
		t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
	}
}
