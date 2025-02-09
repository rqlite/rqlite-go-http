package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_NewClient(t *testing.T) {
	// Create a new HTTP client
	client := NewClient("http://localhost:4001", nil)

	// Check that the client is not nil
	if client == nil {
		t.Error("Expected client to be non-nil")
	}
}

func Test_Client_Close(t *testing.T) {
	// Create a new HTTP client
	client := NewClient("http://localhost:4001", nil)

	// Close the client
	err := client.Close()

	// Check that there was no error
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func Test_BasicAuth(t *testing.T) {
	username := "user"
	password := "pass"

	authExp := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth(username, password)
	authExp = true
	err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth("", "")
	authExp = false
	err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}
