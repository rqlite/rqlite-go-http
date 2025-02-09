package rqlitehttp

import (
	"net/http"
	"time"
)

// DefaultClient returns an HTTP client with a 5-second timeout.
func DefaultClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

// NewTLSClient returns an HTTP client configured for simple TLS, using a CA cert
// and optionally skipping server certificate verification.
func NewTLSClient(caCertPath string, skipVerify bool) (*http.Client, error) {
	// Load CA cert
	// Build *tls.Config
	// Create *http.Transport
	// Return a new *http.Client with that transport
	return &http.Client{}, nil
}

// NewMutualTLSClient returns an HTTP client configured for mutual TLS.
// It accepts paths for the client cert, client key, and trusted CA, plus
// a skipVerify option.
func NewMutualTLSClient(clientCertPath, clientKeyPath, caCertPath string, skipVerify bool) (*http.Client, error) {
	// Load certificates
	// Build *tls.Config with certificate and possibly skipVerify set
	// Create *http.Transport
	// Return a new *http.Client with that transport
	return &http.Client{}, nil
}
