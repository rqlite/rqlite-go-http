package http

import (
	"errors"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

// -----Hosts Tests ----------

func Test_Hosts_ContainsURL(t *testing.T) {
	u1, err := url.Parse("http://host1:4001")
	if err != nil {
		t.Fatal(err)
	}
	u2, err := url.Parse("http://host2:4001")
	if err != nil {
		t.Fatal(err)
	}

	hosts := Hosts{u1}
	if !hosts.ContainsURL(u1) {
		t.Fatal("expected hosts to contain u1")
	}
	if hosts.ContainsURL(u2) {
		t.Fatal("expected hosts not to contain u2")
	}
}

func Test_Hosts_RemoveURL(t *testing.T) {
	u1, err := url.Parse("http://host1:4001")
	if err != nil {
		t.Fatal(err)
	}
	u2, err := url.Parse("http://host2:4001")
	if err != nil {
		t.Fatal(err)
	}
	u3, err := url.Parse("http://host3:4001")
	if err != nil {
		t.Fatal(err)
	}

	hosts := Hosts{u1, u2}
	if !hosts.RemoveURL(u1) {
		t.Fatal("expected RemoveURL to remove u1")
	}
	if hosts.ContainsURL(u1) {
		t.Fatal("expected u1 to be removed")
	}
	if !hosts.ContainsURL(u2) {
		t.Fatal("expected u2 to remain")
	}
	if hosts.RemoveURL(u3) {
		t.Fatal("expected RemoveURL to return false for missing URL")
	}
}

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

func Test_RoundRobinBalancer_CloseIdempotent(t *testing.T) {
	rrb, err := NewRoundRobinBalancer([]string{"http://host1:4001"}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	rrb.Close()
	rrb.Close() // must not panic
}

func Test_RoundRobinBalancer_HealthCheckRestoresHost(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host2:4001"}
	var calls atomic.Int32
	chk := func(u *url.URL) bool {
		calls.Add(1)
		return true
	}
	rrb, err := NewRoundRobinBalancer(urls, chk, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer rrb.Close()

	bad, _ := url.Parse("http://host1:4001")
	rrb.MarkBad(bad)
	if len(rrb.Bad()) != 1 {
		t.Fatalf("expected 1 bad host, got %d", len(rrb.Bad()))
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(rrb.Bad()) == 0 && len(rrb.Healthy()) == 2 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("host was not restored: bad=%d healthy=%d calls=%d",
		len(rrb.Bad()), len(rrb.Healthy()), calls.Load())
}

// -----RandomBalancer Tests ----------

func Test_NewRandomBalancer(t *testing.T) {
	rb, err := NewRandomBalancer([]string{"http://host1:4001", "http://host2:4001"}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()
	if got := len(rb.Healthy()); got != 2 {
		t.Fatalf("expected 2 healthy, got %d", got)
	}
}

func Test_NewRandomBalancer_DuplicateAddresses(t *testing.T) {
	_, err := NewRandomBalancer([]string{"http://host1:4001", "http://host1:4001"}, nil, 0)
	if !errors.Is(err, ErrDuplicateAddresses) {
		t.Fatalf("expected ErrDuplicateAddresses, got %v", err)
	}
}

func Test_RandomBalancer_MarkBad(t *testing.T) {
	urls := []string{"http://host1:4001", "http://host2:4001"}
	rb, err := NewRandomBalancer(urls, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()

	bad, _ := url.Parse("http://host1:4001")
	rb.MarkBad(bad)
	for i := 0; i < 20; i++ {
		u, err := rb.Next()
		if err != nil {
			t.Fatal(err)
		}
		if u.String() == "http://host1:4001" {
			t.Fatalf("host1 was marked bad but still returned")
		}
	}
}

func Test_RandomBalancer_MarkBad_UnknownURL(t *testing.T) {
	rb, err := NewRandomBalancer([]string{"http://host1:4001"}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()

	unknown, _ := url.Parse("http://nope:1")
	rb.MarkBad(unknown) // must not panic
}

func Test_RandomBalancer_NextEmpty(t *testing.T) {
	rb, err := NewRandomBalancer([]string{"http://host1:4001"}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()

	bad, _ := url.Parse("http://host1:4001")
	rb.MarkBad(bad)
	if _, err := rb.Next(); !errors.Is(err, ErrNoHostsAvailable) {
		t.Fatalf("expected ErrNoHostsAvailable, got %v", err)
	}
}

func Test_RandomBalancer_CloseIdempotent(t *testing.T) {
	rb, err := NewRandomBalancer([]string{"http://host1:4001"}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	rb.Close()
	rb.Close() // must not panic
}

func Test_RandomBalancer_HealthCheckRestoresHost(t *testing.T) {
	chk := func(u *url.URL) bool { return true }
	rb, err := NewRandomBalancer([]string{"http://host1:4001"}, chk, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()

	bad, _ := url.Parse("http://host1:4001")
	rb.MarkBad(bad)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(rb.Healthy()) == 1 && len(rb.Bad()) == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("host was not restored: healthy=%d bad=%d", len(rb.Healthy()), len(rb.Bad()))
}
