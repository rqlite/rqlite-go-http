package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_NewClient(t *testing.T) {
	client := NewClient("http://localhost:4001", nil)
	if client == nil {
		t.Error("Expected client to be non-nil")
	}
	if err := client.Close(); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func Test_BasicAuth(t *testing.T) {
	username := "user"
	password := "pass"

	authExp := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !authExp {
			if ok {
				t.Fatalf("basic auth should not be set")
			}
			return
		}

		if !ok {
			t.Fatalf("Expected BasicAuth to be set")
		}
		if exp, got := username, user; exp != got {
			t.Fatalf("Expected user to be '%s', got %s", exp, got)
		}
		if exp, got := password, pass; exp != got {
			t.Fatalf("Expected pass to be '%s', got %s", exp, got)
		}
	}))

	client := NewClient(ts.URL, nil)
	if err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth(username, password)
	authExp = true
	if err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth("", "")
	authExp = false
	if err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}
