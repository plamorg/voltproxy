package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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

	services := map[string]Service{
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

func TestHandlerServiceNotFound(t *testing.T) {
	services := map[string]Service{
		"foo.example.com": {},
	}

	r, w := httptest.NewRequest(http.MethodGet, "http://example.com", nil), httptest.NewRecorder()
	handler(services, false).ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	expectedCode := http.StatusNotFound

	if res.StatusCode != expectedCode {
		t.Errorf("expected code %d got code %d", expectedCode, res.StatusCode)
	}
}

func TestHandlerRedirectToTLS(t *testing.T) {
	r, w := httptest.NewRequest(http.MethodGet, "http://example.com", nil), httptest.NewRecorder()

	services := map[string]Service{
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

func TestTLSHandlerNotFound(t *testing.T) {
	r, w := httptest.NewRequest(http.MethodGet, "http://example.com", nil), httptest.NewRecorder()

	services := map[string]Service{
		"example.com": {
			TLS:    false,
			Router: NewRedirect(url.URL{}),
		},
	}

	// Try access a service through HTTPS when it is specified as non TLS.
	TLSHandler(services).ServeHTTP(w, r)

	expectedCode := http.StatusNotFound

	if w.Code != expectedCode {
		t.Errorf("expected code %d got code %d", expectedCode, w.Code)
	}
}
