package http

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"time"
)

// DefaultClient returns an HTTP client with a 5-second timeout.
func DefaultClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

// NewTLSClientInsecure returns an HTTP client configured for simple TLS, but
// skipping server certificate verification. The client's timeout is
// set as 5 seconds.
func NewTLSClientInsecure() (*http.Client, error) {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}, nil
}

// NewTLSClient returns an HTTP client configured for simple TLS, using the
// provided CA certificate.
func NewTLSClient(caCertPath string) (*http.Client, error) {
	config := &tls.Config{}

	asn1Data, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	config.RootCAs = x509.NewCertPool()
	ok := config.RootCAs.AppendCertsFromPEM(asn1Data)
	if !ok {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
		Timeout: 5 * time.Second,
	}, nil
}

// NewMutualTLSClient returns an HTTP client configured for mutual TLS.
// It accepts paths for the client cert, client key, and trusted CA.
func NewMutualTLSClient(clientCertPath, clientKeyPath, caCertPath string) (*http.Client, error) {
	config := &tls.Config{}

	asn1Data, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	config.RootCAs = x509.NewCertPool()
	ok := config.RootCAs.AppendCertsFromPEM(asn1Data)
	if !ok {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, err
	}
	config.Certificates = []tls.Certificate{cert}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
		Timeout: 5 * time.Second,
	}, nil
}
