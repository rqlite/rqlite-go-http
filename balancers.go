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

	// ErrNoURLSupplied is returned when no host URLs are supplied.
	ErrNoURLSupplied = errors.New("no host URLs supplied")
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
	hosts map[string]*Host

	chkInterval time.Duration
	chckFn      HostChecker
	ch          chan *url.URL

	wg   sync.WaitGroup
	done chan struct{}

	closeOnce sync.Once
}

// Hosts is a slice of host URLs used by balancers to track host state.
type Hosts []*url.URL

// RoundRobinBalancer cycles through healthy hosts in order and can
// optionally restore bad hosts after health checks succeed.
type RoundRobinBalancer struct {
	mu        sync.RWMutex
	goodHosts Hosts
	badHosts  Hosts
	next      uint64

	chckInterval time.Duration
	chckFn       HostChecker
	ch           chan *url.URL

	wg   sync.WaitGroup
	done chan struct{}

	closeOnce sync.Once
}

// NewRoundRobinBalancer returns a RoundRobinBalancer initialized with the
// supplied URLs, health checker, and check interval.
func NewRoundRobinBalancer(urls []string, chckFn HostChecker, d time.Duration) (*RoundRobinBalancer, error) {
	if len(urls) == 0 {
		return nil, ErrNoURLSupplied
	}
	goodHosts := make(Hosts, 0, len(urls))
	for _, s := range urls {
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		if goodHosts.ContainsURL(u) {
			return nil, ErrDuplicateAddresses
		}
		goodHosts = append(goodHosts, u)
	}

	rrb := &RoundRobinBalancer{
		goodHosts:    goodHosts,
		chckInterval: d,
		chckFn:       chckFn,
		done:         make(chan struct{}),
	}
	if chckFn != nil && d > 0 {
		rrb.wg.Go(rrb.checkBadHosts)
	}
	return rrb, nil
}

// Next returns next available healthy Node in Round-Robin Order
func (rrb *RoundRobinBalancer) Next() (*url.URL, error) {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()

	if len(rrb.goodHosts) == 0 {
		return nil, ErrNoHostsAvailable
	}

	idx := (rrb.next) % uint64(len(rrb.goodHosts))
	rrb.next++
	return rrb.goodHosts[idx], nil
}

// MarkBad moves a healthy host to the bad host list so it is skipped until it
// passes health checking again.
func (rrb *RoundRobinBalancer) MarkBad(u *url.URL) {
	rrb.mu.Lock()
	defer rrb.mu.Unlock()

	for i, host := range rrb.goodHosts {
		if host.String() != u.String() {
			continue
		}

		rrb.goodHosts = append(rrb.goodHosts[:i], rrb.goodHosts[i+1:]...)
		if !rrb.badHosts.ContainsURL(host) {
			rrb.badHosts = append(rrb.badHosts, host)
		}
		break
	}
}

// Healthy returns the currently healthy hosts
func (rrb *RoundRobinBalancer) Healthy() []*url.URL {
	rrb.mu.RLock()
	defer rrb.mu.RUnlock()
	healthy := make([]*url.URL, len(rrb.goodHosts))
	copy(healthy, rrb.goodHosts)
	return healthy
}

// Bad returns current bad hosts
func (rrb *RoundRobinBalancer) Bad() []*url.URL {
	rrb.mu.RLock()
	defer rrb.mu.RUnlock()
	bad := make([]*url.URL, len(rrb.badHosts))
	copy(bad, rrb.badHosts)
	return bad
}

// Close stops the balancer's background health checker. It is safe to call
// multiple times. It always returns nil; the error in the signature is provided
// so the type satisfies LoadBalancerCloser.
func (rrb *RoundRobinBalancer) Close() error {
	rrb.closeOnce.Do(func() {
		close(rrb.done)
		rrb.wg.Wait()
	})
	return nil
}

// checkBadHosts periodically checks bad hosts and restores healthy ones to the
// round-robin pool.
func (rrb *RoundRobinBalancer) checkBadHosts() {
	ticker := time.NewTicker(rrb.chckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rrb.mu.RLock()
			bad := make([]*url.URL, len(rrb.badHosts))
			copy(bad, rrb.badHosts)
			rrb.mu.RUnlock()
			for _, host := range bad {
				if !rrb.chckFn(host) {
					continue
				}
				rrb.mu.Lock()
				if rrb.badHosts.RemoveURL(host) && !rrb.goodHosts.ContainsURL(host) {
					rrb.goodHosts = append(rrb.goodHosts, host)
				}
				rrb.mu.Unlock()
			}
		case <-rrb.done:
			return
		}
	}
}

// ContainsURL reports whether the host list contains the target URL.
func (h Hosts) ContainsURL(target *url.URL) bool {
	for _, u := range h {
		if u.String() == target.String() {
			return true
		}
	}
	return false
}

// RemoveURL removes the target URL from the host list and reports whether it
// was found.
func (h *Hosts) RemoveURL(target *url.URL) bool {
	for i, u := range *h {
		if u.String() != target.String() {
			continue
		}
		copy((*h)[i:], (*h)[i+1:])
		(*h)[len(*h)-1] = nil
		*h = (*h)[:len(*h)-1]
		return true
	}
	return false
}

// NewRandomBalancer returns a new RandomBalancer.
func NewRandomBalancer(urls []string, chckFn HostChecker, d time.Duration) (*RandomBalancer, error) {
	hosts := make(map[string]*Host)
	for _, s := range urls {
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		if _, ok := hosts[u.String()]; ok {
			return nil, ErrDuplicateAddresses
		}
		hosts[u.String()] = &Host{URL: u, Healthy: true}
	}
	if len(hosts) == 0 {
		return nil, ErrNoHostsAvailable
	}
	rb := &RandomBalancer{
		hosts:       hosts,
		chkInterval: d,
		chckFn:      chckFn,
		ch:          make(chan *url.URL, len(hosts)),
		done:        make(chan struct{}),
	}

	if chckFn != nil && d > 0 {
		rb.wg.Add(2)
		go rb.checkBadHosts()
		go rb.markGoodHosts()
	}
	return rb, nil
}

// Next returns a random address from the list of addresses it currently
// considers healthy.
func (rb *RandomBalancer) Next() (*url.URL, error) {
	healthy := rb.Healthy()
	if len(healthy) == 0 {
		return nil, ErrNoHostsAvailable
	}
	idx := rand.IntN(len(healthy))
	return healthy[idx], nil
}

// MarkBad marks an address returned by Next() as bad. The RandomBalancer
// will not return this address until the RandomBalancer considers it healthy
// again.
func (rb *RandomBalancer) MarkBad(u *url.URL) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if h, ok := rb.hosts[u.String()]; ok {
		h.Healthy = false
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
// It is safe to call multiple times. It always returns nil; the error in the
// signature is provided so the type satisfies LoadBalancerCloser.
func (rb *RandomBalancer) Close() error {
	rb.closeOnce.Do(func() {
		close(rb.done)
		rb.wg.Wait()
	})
	return nil
}

// checkBadHosts periodically checks unhealthy hosts and queues hosts that have
// become healthy again.
func (rb *RandomBalancer) checkBadHosts() {
	defer rb.wg.Done()
	ticker := time.NewTicker(rb.chkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rb.mu.RLock()
			bad := make([]*url.URL, 0)
			for _, host := range rb.hosts {
				if !host.Healthy {
					bad = append(bad, host.URL)
				}
			}
			rb.mu.RUnlock()

			for _, u := range bad {
				if !rb.chckFn(u) {
					continue
				}
				select {
				case rb.ch <- u:
				case <-rb.done:
					return
				}
			}
		case <-rb.done:
			return
		}
	}
}

// markGoodHosts marks hosts received from the health-check channel as healthy
// again.
func (rb *RandomBalancer) markGoodHosts() {
	defer rb.wg.Done()
	for {
		select {
		case u := <-rb.ch:
			rb.mu.Lock()
			if host, ok := rb.hosts[u.String()]; ok {
				host.Healthy = true
			}
			rb.mu.Unlock()
		case <-rb.done:
			return
		}
	}
}
