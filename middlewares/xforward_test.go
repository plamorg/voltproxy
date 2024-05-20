package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestXForwardDisable(t *testing.T) {
	x := XForward{
		Enable: false,
	}

	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range xForwardedHeaders {
			if r.Header.Get(header) != "" {
				t.Errorf("expected empty header %s, got %s", header, r.Header.Get(header))
			}
		}
	})

	handler := x.Handle(server)

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestXForwardEnable(t *testing.T) {
	x := XForward{
		Enable: true,
	}

	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range xForwardedHeaders {
			if r.Header.Get(header) == "" {
				t.Errorf("expected non-empty header %s", header)
			}
		}
	})

	handler := x.Handle(server)

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
}
