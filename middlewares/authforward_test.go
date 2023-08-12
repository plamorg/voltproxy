package middlewares

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
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
			a := AuthForward{
				Address: test.authServer.URL,
			}
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
	const testValue = "test"

	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Test-Header") != testValue {
			t.Errorf("test header was not forwarded")
		}
		if r.Header.Get("Not-Forwarded") != "" {
			t.Errorf("header was unexpectedly forwarded")
		}

		w.WriteHeader(http.StatusOK)
	}))

	a := AuthForward{
		Address:        verifyingAuthServer.URL,
		RequestHeaders: []string{"test-header"},
	}

	handler := a.Handle(teapotHandler)

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Test-Header", testValue)
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

	a := AuthForward{
		Address: verifyingAuthServer.URL,
	}

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

	a := AuthForward{
		Address:         verifyingAuthServer.URL,
		ResponseHeaders: []string{"test-header"},
	}

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

func TestAuthForwardHandleInvalidAddress(t *testing.T) {
	a := AuthForward{
		Address: ":invalid",
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	a.Handle(teapotHandler).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestAuthForwardHandleXForwardedFalse(t *testing.T) {
	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for header, values := range r.Header {
			if strings.HasPrefix(header, "X-Forwarded") {
				t.Errorf("header %s was unexpectedly forwarded with values %+v", header, values)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))

	a := AuthForward{
		Address:    verifyingAuthServer.URL,
		XForwarded: false,
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(xForwardedFor, "foo xForwardedFor")
	r.Header.Set(xForwardedMethod, "bar xForwardedMethod")
	r.Header.Set(xForwardedProto, "baz xForwardedProto")
	r.Header.Set(xForwardedHost, "foobar xForwardedHost")
	r.Header.Set(xForwardedURI, "foobarbaz xForwardedURI")
	a.Handle(teapotHandler).ServeHTTP(w, r)
}

func TestAuthForwardHandleXForwardedTrue(t *testing.T) {
	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tests := []struct {
			header   string
			expected string
		}{
			{xForwardedFor, "1.1.1.1"},
			{xForwardedMethod, "POST"},
			{xForwardedProto, "https"},
			{xForwardedHost, "example.com"},
			{xForwardedURI, "/foobarbaz"},
		}

		for _, test := range tests {
			if r.Header.Get(test.header) != test.expected {
				t.Errorf("expected %s to be forwarded as %s, got %s", test.header, test.expected, r.Header.Get(test.header))
			}
		}
		w.WriteHeader(http.StatusOK)
	}))

	a := AuthForward{
		Address:    verifyingAuthServer.URL,
		XForwarded: true,
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)

	// Manually set X-Forwarded headers should be overwritten.
	r.Header.Set(xForwardedFor, "nonsense")
	r.Header.Set(xForwardedMethod, "nonsense")
	r.Header.Set(xForwardedProto, "nonsense")
	r.Header.Set(xForwardedHost, "nonsense")
	r.Header.Set(xForwardedURI, "nonsense")

	r.RemoteAddr = "1.1.1.1:1234"
	r.Method = "POST"
	r.TLS = &tls.ConnectionState{}
	r.Host = "example.com"
	r.RequestURI = "/foobarbaz"

	a.Handle(teapotHandler).ServeHTTP(w, r)
}

func TestAuthForwardHandleXForwardedProto(t *testing.T) {
	verifyingAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto := r.Header.Get(xForwardedProto)
		if proto != "http" {
			t.Errorf("expected %s to be forwarded as %s, got %s", xForwardedProto, "http", proto)
		}
	}))

	a := AuthForward{
		Address:    verifyingAuthServer.URL,
		XForwarded: true,
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	r.TLS = nil

	a.Handle(teapotHandler).ServeHTTP(w, r)
}

func TestAuthForwardHandleRedirectingAuthServer(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	authRedirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Do-Not-Forward", "foo")
		http.Redirect(w, r, authServer.URL, http.StatusMovedPermanently)
	}))

	a := AuthForward{
		Address: authRedirectServer.URL,
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	a.Handle(teapotHandler).ServeHTTP(w, r)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected %d, got %d", http.StatusMovedPermanently, w.Code)
	}

	location := w.Header().Get("Location")
	if location != authServer.URL {
		t.Errorf("expected Location header to be %s, got %s", authServer.URL, location)
	}

	if w.Header().Get("Do-Not-Forward") != "" {
		t.Errorf("header from auth redirect server was unexpectedly forwarded")
	}
}

func TestAuthForwardHandleBadAuthServer(t *testing.T) {
	badAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "invalid URL :(")
		w.WriteHeader(http.StatusMovedPermanently)
	}))

	a := AuthForward{
		Address: badAuthServer.URL,
	}

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	a.Handle(teapotHandler).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
