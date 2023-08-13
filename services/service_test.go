package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type MockService struct {
	data Data
}

func (m *MockService) Data() Data {
	return m.data
}

func (m *MockService) Remote(_ http.ResponseWriter, _ *http.Request) (*url.URL, error) {
	return nil, fmt.Errorf("something bad happened")
}

func TestFindServiceWithHostFailure(t *testing.T) {
	tests := map[string]struct {
		list          List
		host          string
		expectedError error
	}{
		"empty list": {
			list:          List{},
			host:          "example.com",
			expectedError: errNoServiceFound,
		},
		"not found": {
			list: List{
				NewRedirect(Data{Host: "sub.example.com"}, "https://example.com"),
			},
			host:          "example.com",
			expectedError: errNoServiceFound,
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
	expectedService := NewRedirect(Data{Host: "example.com"}, "https://example.com")

	list := List{
		NewRedirect(Data{Host: "foo.example.com"}, "https://example.com"),
		expectedService,
		NewRedirect(Data{Host: "bar.example.com"}, "https://foo.example.com"),
	}

	service, err := list.findServiceWithHost("example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(*service, expectedService) {
		t.Errorf("expected %v got %v", expectedService, *service)
	}
}

func TestHandlerSuccess(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	w := httptest.NewRecorder()

	list := List{
		NewRedirect(Data{Host: "sub.example.com"}, "https://bar.example.com"),
		NewRedirect(Data{Host: "example.com"}, okServer.URL),
	}

	expectedHost := strings.Split(okServer.URL, "://")[1]

	list.Handler().ServeHTTP(w, r)
	if r.Host != expectedHost {
		t.Errorf("expected host %s got host %s", expectedHost, r.Host)
	}
}

func TestHandlerRedirectToTLS(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	w := httptest.NewRecorder()

	list := List{
		NewRedirect(Data{Host: "example.com", TLS: true}, "https://bar.example.com"),
	}

	// Access a TLS service through HTTP and expect to get redirected to HTTPS.
	list.Handler().ServeHTTP(w, r)

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
	r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	w := httptest.NewRecorder()

	list := List{
		NewRedirect(Data{Host: "example.com", TLS: false}, "https://baz.example.com"),
	}

	// Try access a service through HTTPS when it is specified as non TLS.
	list.TLSHandler().ServeHTTP(w, r)

	expectedCode := http.StatusNotFound

	if w.Code != expectedCode {
		t.Errorf("expected code %d got code %d", expectedCode, w.Code)
	}
}
