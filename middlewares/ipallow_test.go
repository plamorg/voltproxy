package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestIpAllowedEmptyList(t *testing.T) {
	ipAllow := NewIPAllow([]string{})

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.0.1.0"

	w := httptest.NewRecorder()
	ipAllow.Handle(okHandler).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status code %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestIpAllowHandle(t *testing.T) {
	ipAllow := NewIPAllow([]string{
		"172.0.0.1",
		"192.168.0.0/24",
		"10.9.0.1",
	})

	tests := map[string]struct {
		remoteAddr     string
		expectedStatus int
	}{
		"allowed": {
			remoteAddr:     "10.9.0.1:12345",
			expectedStatus: http.StatusOK,
		},
		"not allowed": {
			remoteAddr:     "172.0.0.2:1234",
			expectedStatus: http.StatusForbidden,
		},
		"allowed through CIDR": {
			remoteAddr:     "192.168.0.31:3000",
			expectedStatus: http.StatusOK,
		},
		"not allowed through CIDR": {
			remoteAddr:     "192.168.1.0:1234",
			expectedStatus: http.StatusForbidden,
		},
		"disallow no port specified": {
			remoteAddr:     "172.0.0.1",
			expectedStatus: http.StatusForbidden,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = test.remoteAddr

			w := httptest.NewRecorder()
			ipAllow.Handle(okHandler).ServeHTTP(w, r)

			if status := w.Code; status != test.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, test.expectedStatus)
			}
		})
	}
}

func TestInCIDR(t *testing.T) {
	tests := map[string]struct {
		ip       string
		cidr     string
		expected bool
	}{
		"subnet mask 32 true": {
			ip:       "172.1.2.4",
			cidr:     "172.1.2.4/32",
			expected: true,
		},
		"subnet mask 32 false": {
			ip:       "192.168.0.0",
			cidr:     "192.168.0.1/32",
			expected: false,
		},
		"in CIDR": {
			ip:       "10.1.2.254",
			cidr:     "10.1.2.0/24",
			expected: true,
		},
		"not in CIDR": {
			ip:       "192.168.4.255",
			cidr:     "192.168.0.0/22",
			expected: false,
		},
		"not a valid CIDR": {
			ip:       "172.0.0.4",
			cidr:     "172.0.0.4",
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if result := inCIDR(test.ip, test.cidr); result != test.expected {
				t.Errorf("expected %s in %s to be %v, got %v", test.ip, test.cidr, test.expected, result)
			}
		})
	}
}
