package http

import (
	"errors"
	"math/rand/v2"
	"net/url"
	"sync"
	"time"
)

var (
	// ErrNoHostsAvailable is returned when no hosts are available.
	ErrNoHostsAvailable = errors.New("no hosts available")

	// ErrDuplicateAddresses is returned when duplicate addresses are provided
	// to a balancer.
	ErrDuplicateAddresses = errors.New("duplicate addresses provided")
)

// LoopbackBalancer takes a single address and always returns it when Next() is called.
// It performs no healthchecking.
type LoopbackBalancer struct {
	u *url.URL
}

// NewLoopbackBalancer returns a new LoopbackBalancer.
func NewLoopbackBalancer(address string) (*LoopbackBalancer, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	return &LoopbackBalancer{
		u: u,
	}, nil
}

// Next returns the next address in the list of addresses.
func (lb *LoopbackBalancer) Next() (*url.URL, error) {
	return lb.u, nil
}

// Host represents a URL and its health status.
type Host struct {
	URL     *url.URL
	Healthy bool
}

// HostChecker is a function that takes a URL and returns true if the URL is
// healthy.
type HostChecker func(url *url.URL) bool

// RandomBalancer takes a list of addresses and returns a random one from its
// healthy list when Next() is called. At the start all supplied addresses are
// considered healthy. If a client detects that an address is unhealthy, it can
// call MarkBad() to mark the address as unhealthy. The RandomBalancer will
// then periodically check the health of the address and mark it as healthy
// again if and when it becomes healthy.
type RandomBalancer struct {
	mu    sync.RWMutex
	hosts []*Host

	chkInterval time.Duration
	chckFn      HostChecker
	ch          chan *url.URL

	wg   sync.WaitGroup
	done chan struct{}
}

// NewRandomBalancer returns a new RandomBalancer.
func NewRandomBalancer(addresses []string, chckFn HostChecker, d time.Duration) (*RandomBalancer, error) {
	hosts := make([]*Host, 0, len(addresses))
	seen := make(map[string]struct{})
	for _, s := range addresses {
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[u.String()]; ok {
			return nil, ErrDuplicateAddresses
		}
		seen[u.String()] = struct{}{}
		hosts = append(hosts, &Host{URL: u, Healthy: true})
	}
	if len(hosts) == 0 {
		return nil, ErrNoHostsAvailable
	}
	rb := &RandomBalancer{
		hosts:       hosts,
		chkInterval: d,
		chckFn:      chckFn,
	}

	rb.wg.Add(2)
	go rb.checkBadHosts()
	go rb.markGoodHosts()
	return rb, nil
}

// Next returns a random address from the list of addresses it currently
// considers healthy.
func (rb *RandomBalancer) Next() (*url.URL, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var healthy []*Host
	for _, host := range rb.hosts {
		if host.Healthy {
			healthy = append(healthy, host)
		}
	}

	if len(healthy) == 0 {
		return nil, ErrNoHostsAvailable
	}
	idx := rand.IntN(len(healthy))
	return healthy[idx].URL, nil
}

// MarkBad marks an address returned by Next() as bad. The RandomBalancer
// will not return this address until the RandomBalancer considers it healthy
// again.
func (rb *RandomBalancer) MarkBad(u *url.URL) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	for _, host := range rb.hosts {
		if host.URL.String() == u.String() {
			host.Healthy = false
			return
		}
	}
}

// Healthy returns the slice of currently healthy hosts.
func (rb *RandomBalancer) Healthy() []*url.URL {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var healthy []*url.URL
	for _, host := range rb.hosts {
		if host.Healthy {
			healthy = append(healthy, host.URL)
		}
	}
	return healthy
}

// Bad returns the slice of currently bad hosts.
func (rb *RandomBalancer) Bad() []*url.URL {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var bad []*url.URL
	for _, host := range rb.hosts {
		if !host.Healthy {
			bad = append(bad, host.URL)
		}
	}
	return bad
}

// Close closes the RandomBalancer. A closed RandomBalancer should not be reused.
func (rb *RandomBalancer) Close() {
	close(rb.done)
	rb.wg.Wait()
}

func (rb *RandomBalancer) checkBadHosts() {
	defer rb.wg.Done()
	ticker := time.NewTicker(rb.chkInterval)
	for {
		select {
		case <-ticker.C:
			rb.mu.RLock()
			for _, host := range rb.hosts {
				if !host.Healthy {
					if ok := rb.chckFn(host.URL); ok {
						rb.ch <- host.URL
					}
				}
			}
			rb.mu.RUnlock()
		case <-rb.done:
			return
		}
	}
}

func (rb *RandomBalancer) markGoodHosts() {
	defer rb.wg.Done()
	for {
		select {
		case u := <-rb.ch:
			rb.mu.Lock()
			for _, host := range rb.hosts {
				if host.URL == u {
					host.Healthy = true
					break
				}
			}
			rb.mu.Unlock()
		}
	}
}
