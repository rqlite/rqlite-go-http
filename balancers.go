package http

import (
	"errors"
	"net/url"
)

var (
	ErrNoHostsAvailable = errors.New("no hosts available")

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
func (rb *LoopbackBalancer) Next() (*url.URL, error) {
	return rb.u, nil
}
