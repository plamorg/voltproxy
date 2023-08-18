package health

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAlwaysUp(t *testing.T) {
	au := Always(true)
	au.Launch(func(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
		return nil, nil
	})
	if au.Up() != true {
		t.Errorf("expected always up to be healthy")
	}
	if au.Check() != nil {
		t.Errorf("expected always up channel to be nil")
	}
}

func TestAlwaysDown(t *testing.T) {
	au := Always(false)
	au.Launch(func(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
		return nil, nil
	})
	if au.Up() != false {
		t.Errorf("expected always up to be unhealthy")
	}
	if au.Check() != nil {
		t.Errorf("expected always up channel to be nil")
	}
}
