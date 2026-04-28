package http

import (
	"errors"
	"net/url"
	"testing"
)

// -----RoundRobinBalancer Tests ----------

func Test_NewRoundRobinBalancer(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host2:4002", "http://host3:4003"}

	rrb, err := NewRoundRobinBalancer(urls, nil, 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer rrb.Close()
	if rrb == nil {
		t.Fatalf("expected non-nil balancer, got nil")
	}
}

func Test_NewRoundRobinBalancer_EmptyURLs(t *testing.T) {
	_, err := NewRoundRobinBalancer([]string{}, nil, 0)
	if !errors.Is(err, ErrNoURLSupplied) {
		t.Fatalf("expected ErrHostsAvailable,got %v", err)
	}
}

func Test_NewRoundRobinBalancer_DuplicateAddresses(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host1:4001"}
	_, err := NewRoundRobinBalancer(urls, nil, 0)
	if !errors.Is(err, ErrDuplicateAddresses) {
		t.Fatalf("expected ErrDuplicateAddresses,got %v", err)
	}
}

func Test_RoundRobinOrder(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host2:4002",
		"http://host3:4003"}

	rrb, err := NewRoundRobinBalancer(urls, nil, 0)
	if err != nil {
		t.Fatalf("expected nil error,got %v", err)
	}
	defer rrb.Close()

	for i := 0; i < len(urls); i++ {
		u, err := rrb.Next()
		if err != nil {
			t.Fatalf("expected nil error,got %v", err)
		}
		if u.String() != urls[i] {
			t.Fatalf("expected %s,got %s", urls[i], u.String())
		}
	}
}

func Test_RoundRobinBalancer_MarkBad(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host2:4002"}
	rrb, err := NewRoundRobinBalancer(urls, nil, 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer rrb.Close()
	badURL, _ := url.Parse("http://host1:4001")
	rrb.MarkBad(badURL)
	// After marking host1 bad, Next() should only return host2
	for i := 0; i < 5; i++ {
		u, err := rrb.Next()
		if err != nil {
			t.Fatalf("iteration %d: expected nil error, got %v", i, err)
		}
		if u.String() == "http://host1:4001" {
			t.Fatalf("iteration %d: host1 was marked bad but still returned", i)
		}
	}
}

func Test_RoundRobinBalancer_Healthy(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host2:4001", "http://host3:4001"}
	rrb, err := NewRoundRobinBalancer(urls, nil, 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer rrb.Close()
	healthy := rrb.Healthy()
	if len(healthy) == 0 {
		t.Fatal("expected at least one healthy host")
	}
	// Mark one bad and check again
	badURL, _ := url.Parse("http://host1:4001")
	rrb.MarkBad(badURL)
	healthy = rrb.Healthy()
	for _, h := range healthy {
		if h != nil && h.String() == "http://host1:4001" {
			t.Fatal("host1 was marked bad but still appears in Healthy()")
		}
	}
}
