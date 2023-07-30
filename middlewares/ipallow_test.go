package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIpAllowHandle(t *testing.T) {
	ipAllow := NewIPAllow([]string{"172.0.0.1", "10.9.0.1"})

	handler := ipAllow.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := map[string]struct {
		remoteAddr     string
		expectedStatus int
	}{
		"allowed": {
			remoteAddr:     "10.9.0.1:12345",
			expectedStatus: http.StatusOK,
		},
		"not allowed": {
			remoteAddr:     "172.0.0.2",
			expectedStatus: http.StatusForbidden,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = test.remoteAddr

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if status := w.Code; status != test.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, test.expectedStatus)
			}
		})
	}
}
