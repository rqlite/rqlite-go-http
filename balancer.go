package http

import (
	"errors"
	"sync"
)

var (
	ErrNoAddresses = errors.New("no addresses available")

	ErrDuplicateAddresses = errors.New("duplicate addresses provided")
)

// Balancer is an interface for selecting which node (address) a request
// should be sent to, plus receiving feedback about whether that request
// ultimately succeeded or failed.
//
// You can implement your own Balancer to incorporate custom logic like
// reading from the nearest node, writing to the leader, or distributing
// load across nodes with specialized heuristics.
type Balancer interface {
	// Next returns the next address to try. An implementation might return an
	// error if there are no addresses available, or if all known addresses have
	// recently failed, etc.
	//
	// The method could also accept arguments about the request (e.g. read vs.
	// write) if you want to route them differently.
	Next() (string, error)

	// MarkSuccess informs the Balancer that a request to the specified address
	// succeeded, allowing the balancer to update any internal metrics, counters,
	// or tracking that determines future selections. If addr was not previously
	// known to the balancer, the call does nothing.
	MarkSuccess(addr string)

	// MarkFailure informs the Balancer that a request to the specified address
	// failed, letting the balancer mark the address as unhealthy or increment
	// an error count. If addr was not previously known to the balancer, the call
	// does nothing.
	MarkFailure(addr string)
}

// RandomBalancer picks a random address from the list for each call to Next.
type RandomBalancer struct {
	addresses    map[string]bool
	numAvailable int
	mu           sync.RWMutex
}

// NewRandomBalancer initializes a RandomBalancer given a non-empty list of addresses.
func NewRandomBalancer(addresses []string) (*RandomBalancer, error) {
	if len(addresses) == 0 {
		return nil, errors.New("must provide at least one address")
	}

	// Any duplicate addresses would be a mistake in the configuration.
	a := make(map[string]bool, len(addresses))
	for _, addr := range addresses {
		if _, ok := a[addr]; ok {
			return nil, ErrDuplicateAddresses
		}
		a[addr] = true
	}

	rb := &RandomBalancer{
		addresses:    a,
		numAvailable: len(a),
	}
	return rb, nil
}

// Next returns one of the known addresses at random.
func (rb *RandomBalancer) Next() (string, error) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.numAvailable == 0 {
		return "", errors.New("no addresses available")
	}

	// return a random address - mock for now
	return "", nil
}

// MarkSuccess marks the specified address as healthy.
func (rb *RandomBalancer) MarkSuccess(addr string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	a, ok := rb.addresses[addr]
	if a || !ok {
		return
	}
	rb.addresses[addr] = true
	rb.numAvailable++
}

// MarkFailure marks the specified address as unhealthy.
func (rb *RandomBalancer) MarkFailure(addr string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	a, ok := rb.addresses[addr]
	if !a || !ok {
		return
	}
	rb.addresses[addr] = false
	rb.numAvailable--
}
