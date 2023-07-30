package services

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestFindServiceWithHostFailure(t *testing.T) {
	tests := map[string]struct {
		list          List
		host          string
		expectedError error
	}{
		"empty list": {
			list:          List{},
			host:          "example.com",
			expectedError: ErrNoServiceFound,
		},
		"not found": {
			list: List{
				NewRedirect("sub.example.com", nil, "https://example.com"),
			},
			host:          "example.com",
			expectedError: ErrNoServiceFound,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := test.list.findServiceWithHost(test.host)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("expected error %v got error %v", test.expectedError, err)
			}
		})
	}
}

func TestFindServiceWithHostSuccess(t *testing.T) {
	expectedService := NewRedirect("example.com", nil, "https://example.com")

	list := List{
		NewRedirect("foo.example.com", nil, "https://example.com"),
		expectedService,
		NewRedirect("bar.example.com", nil, "https://foo.example.com"),
	}

	service, err := list.findServiceWithHost("example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if (*service).Host() != expectedService.Host() {
		t.Errorf("expected %s got %s", expectedService.Host(), (*service).Host())
	}
}

func TestProxySuccess(t *testing.T) {
	r := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()

	list := List{
		NewRedirect("sub.example.com", nil, "https://bar.example.com"),
		NewRedirect("example.com", nil, "https://foo.example.com"),
	}

	expectedHost := "foo.example.com"

	list.Proxy(r, w)
	if r.Host != expectedHost {
		t.Errorf("expected host %s got host %s", expectedHost, r.Host)
	}
}
