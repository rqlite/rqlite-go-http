package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultHTTPClient returns an HTTP client with a 5-second timeout.
func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

// NewHTTPTLSClientInsecure returns an HTTP client configured for simple TLS, but
// skipping server certificate verification. The client's timeout is
// set as 5 seconds.
func NewHTTPTLSClientInsecure() (*http.Client, error) {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}, nil
}

// NewHTTPTLSClient returns an HTTP client configured for simple TLS, using the
// provided CA certificate.
func NewHTTPTLSClient(caCertPath string) (*http.Client, error) {
	config := &tls.Config{}

	asn1Data, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	config.RootCAs = x509.NewCertPool()
	ok := config.RootCAs.AppendCertsFromPEM(asn1Data)
	if !ok {
		return nil, fmt.Errorf("failed to append CA certs from PEM")
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
		Timeout: 5 * time.Second,
	}, nil
}

// NewHTTPMutualTLSClient returns an HTTP client configured for mutual TLS.
// It accepts paths for the client cert, client key, and trusted CA.
func NewHTTPMutualTLSClient(clientCertPath, clientKeyPath, caCertPath string) (*http.Client, error) {
	config := &tls.Config{}

	asn1Data, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	config.RootCAs = x509.NewCertPool()
	ok := config.RootCAs.AppendCertsFromPEM(asn1Data)
	if !ok {
		return nil, fmt.Errorf("failed to append CA certs from PEM")
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

// ExecuteResponse represents the JSON returned by /db/execute.
type ExecuteResponse struct {
	Results        []ExecuteResult `json:"results"`
	Time           float64         `json:"time,omitempty"`
	Error          string          `json:"error,omitempty"`
	SequenceNumber int64           `json:"sequence_number,omitempty"`
	RaftIndex      int64           `json:"raft_index,omitempty"`
}

// HasError returns true if any of the results in the response contain an error.
// If an error is found, the index of the result and the error message are returned.
func (er *ExecuteResponse) HasError() (bool, int, string) {
	if er.Error != "" {
		return true, -1, er.Error
	}

	for i, result := range er.Results {
		if result.Error != "" {
			return true, i, result.Error
		}
	}
	return false, -1, ""
}

// ExecuteResult is an element of ExecuteResponse.Results.
type ExecuteResult struct {
	LastInsertID int64   `json:"last_insert_id"`
	RowsAffected int64   `json:"rows_affected"`
	Time         float64 `json:"time,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// QueryResponse represents the JSON returned by /db/query in the default (non-associative) form.
type QueryResponse struct {
	Results   any     `json:"results"`
	Time      float64 `json:"time,omitempty"`
	Error     string  `json:"error,omitempty"`
	RaftIndex int64   `json:"raft_index,omitempty"`
}

// QueryResult is an element of QueryResponse.Results.
type QueryResult struct {
	Columns []string `json:"columns"`
	Types   []string `json:"types"`
	Values  [][]any  `json:"values"`
	Time    float64  `json:"time,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// QueryResultAssoc is an element of QueryResponse.Results, but in an associative form.
type QueryResultAssoc struct {
	Types map[string]string `json:"types"`
	Rows  []map[string]any  `json:"rows"`
	Time  float64           `json:"time,omitempty"`
	Error string            `json:"error,omitempty"`
}

// HasError returns true if any of the results in the response contain an error.
// If an error is found, the index of the result and the error message are returned.
func (qr *QueryResponse) HasError() (bool, int, string) {
	if qr.Error != "" {
		return true, -1, qr.Error
	}

	switch v := qr.Results.(type) {
	case []QueryResult:
		for i, result := range v {
			if result.Error != "" {
				return true, i, result.Error
			}
		}
	case []QueryResultAssoc:
		for i, result := range v {
			if result.Error != "" {
				return true, i, result.Error
			}
		}
	}
	return false, -1, ""
}

// GetQueryResults returns the results as a slice of QueryResult. This can be convenient
// when the caller knows the type of the results in advance. If the results are not a
// slice of QueryResult, a panic will occur.
func (qr *QueryResponse) GetQueryResults() []QueryResult {
	return qr.Results.([]QueryResult)
}

// GetQueryResultsAssoc returns the results as a slice of QueryResultAssoc. This can be
// convenient when the caller knows the type of the results in advance. If the results
// are not a slice of QueryResultAssoc, a panic will occur.
func (qr *QueryResponse) GetQueryResultsAssoc() []QueryResultAssoc {
	return qr.Results.([]QueryResultAssoc)
}

// UnmarshalJSON implements the json.Unmarshaler interface for QueryResponse.
func (qr *QueryResponse) UnmarshalJSON(data []byte) error {
	// Define an alias to avoid recursion.
	type Alias QueryResponse
	aux := &struct {
		Results json.RawMessage `json:"results"`
		*Alias
	}{
		Alias: (*Alias)(qr),
	}

	// Unmarshal into the auxiliary struct.
	auxDec := json.NewDecoder(bytes.NewReader(data))
	auxDec.UseNumber()
	if err := auxDec.Decode(aux); err != nil {
		return err
	}

	var res []QueryResult
	resDec := json.NewDecoder(bytes.NewReader(aux.Results))
	resDec.UseNumber()
	if err := resDec.Decode(&res); err == nil {
		qr.Results = res
		return nil
	}

	var resAssoc []QueryResultAssoc
	resAssocDec := json.NewDecoder(bytes.NewReader(aux.Results))
	resAssocDec.UseNumber()
	if err := resAssocDec.Decode(&resAssoc); err == nil {
		qr.Results = resAssoc
		return nil
	}

	return fmt.Errorf("unable to unmarshal results into either []QueryResult or []QueryResultAssoc")
}

// RequestResponse represents the JSON returned by /db/request.
type RequestResponse struct {
	Results   any     `json:"results"`
	Time      float64 `json:"time,omitempty"`
	Error     string  `json:"error,omitempty"`
	RaftIndex int64   `json:"raft_index,omitempty"`
}

// RequestResult is an element of RequestResponse.Results.
// It may include either Query-like results, Execute-like results, or both.
type RequestResult struct {
	// Same fields as QueryResult plus ExecuteResult fields.
	// If read-only, LastInsertID and RowsAffected would be empty;
	// if write-only, Columns and Values would be empty.
	Columns      []string `json:"columns"`
	Types        []string `json:"types"`
	Values       [][]any  `json:"values"`
	LastInsertID *int64   `json:"last_insert_id"`
	RowsAffected *int64   `json:"rows_affected"`
	Error        string   `json:"error,omitempty"`
	Time         float64  `json:"time,omitempty"`
}

// RequestResultAssoc is an element of RequestResponse.Results, but in an associative form.
// It may include Query-like results, Execute-like results, or both.
type RequestResultAssoc struct {
	Types        map[string]string `json:"types"`
	Rows         []map[string]any  `json:"rows"`
	LastInsertID *int64            `json:"last_insert_id"`
	RowsAffected *int64            `json:"rows_affected"`
	Error        string            `json:"error,omitempty"`
	Time         float64           `json:"time,omitempty"`
}

// GetRequestResults returns the results as a slice of RequestResult. This can be convenient
// when the caller does not know the type of the results in advance. If the results are not
// a slice of RequestResult, a panic will occur.
func (rr *RequestResponse) GetRequestResults() []RequestResult {
	return rr.Results.([]RequestResult)
}

// GetRequestResultsAssoc returns the results as a slice of RequestResultAssoc. This can be
// convenient when the caller does not know the type of the results in advance. If the results
// are not a slice of RequestResultAssoc, a panic will occur.
func (rr *RequestResponse) GetRequestResultsAssoc() []RequestResultAssoc {
	return rr.Results.([]RequestResultAssoc)
}

// HasError returns true if any of the results in the response contain an error.
// If an error is found, the index of the result and the error message are returned.
func (rr *RequestResponse) HasError() (bool, int, string) {
	if rr.Error != "" {
		return true, -1, rr.Error
	}

	switch v := rr.Results.(type) {
	case []RequestResult:
		for i, result := range v {
			if result.Error != "" {
				return true, i, result.Error
			}
		}
	case []RequestResultAssoc:
		for i, result := range v {
			if result.Error != "" {
				return true, i, result.Error
			}
		}
	}
	return false, -1, ""
}

// UnmarshalJSON implements the json.Unmarshaler interface for RequestResponse.
func (qr *RequestResponse) UnmarshalJSON(data []byte) error {
	// Define an alias to avoid recursion.
	type Alias RequestResponse
	aux := &struct {
		Results json.RawMessage `json:"results"`
		*Alias
	}{
		Alias: (*Alias)(qr),
	}

	// Unmarshal into the auxiliary struct.
	auxDec := json.NewDecoder(bytes.NewReader(data))
	auxDec.UseNumber()
	if err := auxDec.Decode(aux); err != nil {
		return err
	}

	var res []RequestResult
	resDec := json.NewDecoder(bytes.NewReader(aux.Results))
	resDec.UseNumber()
	if err := resDec.Decode(&res); err == nil {
		qr.Results = res
		return nil
	}

	var resAssoc []RequestResultAssoc
	resAssocDec := json.NewDecoder(bytes.NewReader(aux.Results))
	resAssocDec.UseNumber()
	if err := resAssocDec.Decode(&resAssoc); err == nil {
		qr.Results = resAssoc
		return nil
	}

	return fmt.Errorf("unable to unmarshal results into either []RequestResult or []RequestResultAssoc")
}

const (
	executePath = "/db/execute"
	queryPath   = "/db/query"
	requestPath = "/db/request"
	backupPath  = "/db/backup"
	loadPath    = "/db/load"
	bootPath    = "/boot"
	statusPath  = "/status"
	expvarPath  = "/debug/vars"
	nodesPath   = "/nodes"
	readyPath   = "/readyz"
	removePath  = "/remove"
)

// LoadBalancer is the interface load balancers must support.
type LoadBalancer interface {
	// Next returns the next URL to use for the request.
	Next() (*url.URL, error)
}

// Client is the main type through which rqlite is accessed.
type Client struct {
	lb         LoadBalancer
	httpClient *http.Client

	promoteErrors atomic.Bool

	mu            sync.RWMutex
	basicAuthUser string
	basicAuthPass string
}

// NewClient creates a new Client with default settings. If httpClient is nil,
// the the default client is used.
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	lb, err := NewLoopbackBalancer(baseURL)
	if err != nil {
		return nil, err
	}

	cl := &Client{
		lb:         lb,
		httpClient: httpClient,
	}
	if cl.httpClient == nil {
		cl.httpClient = DefaultHTTPClient()
	}
	return cl, nil
}

// SetBasicAuth configures the client to use Basic Auth for all subsequent requests.
// Pass empty strings to disable Basic Auth.
func (c *Client) SetBasicAuth(username, password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.basicAuthUser = username
	c.basicAuthPass = password
}

// PromoteErrors enables or disables the promotion of statement-level errors to Go errors.
//
// By default an operation on the client only returns an error if there is a failure at
// the HTTP level and it is up to the caller to inspect the response body for statement-level
// errors.
//
// However if this method is called with true, then the client will also inspect the response
// body and return an error if there is any failure at the statement level, setting the returned
// error to the first statement-level error encountered.
func (c *Client) PromoteErrors(b bool) {
	c.promoteErrors.Store(b)
}

// ExecuteSingle performs a single write operation (INSERT, UPDATE, DELETE) using /db/execute.
// args should be a single map of named parameters, or a slice of positional parameters.
// It is the caller's responsibility to ensure the correct number and type of parameters.
func (c *Client) ExecuteSingle(ctx context.Context, statement string, args ...any) (*ExecuteResponse, error) {
	stmt, err := NewSQLStatement(statement, args...)
	if err != nil {
		return nil, err
	}
	return c.Execute(ctx, SQLStatements{stmt}, nil)
}

// Execute executes one or more SQL statements (INSERT, UPDATE, DELETE) using /db/execute.
// opts may be nil, in which case default options are used.
func (c *Client) Execute(ctx context.Context, statements SQLStatements, opts *ExecuteOptions) (retEr *ExecuteResponse, retErr error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	queryParams, err := makeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONPostRequest(ctx, executePath, queryParams, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	}

	var executeResp ExecuteResponse
	execRespDec := json.NewDecoder(bytes.NewReader(respBody))
	execRespDec.UseNumber()
	if err := execRespDec.Decode(&executeResp); err != nil {
		return nil, err
	}

	if c.promoteErrors.Load() {
		if f, i, msg := executeResp.HasError(); f {
			retErr = fmt.Errorf("statement %d: %s", i, msg)
		}
	}
	return &executeResp, retErr
}

// QuerySingle performs a single read operation (SELECT) using /db/query.
// args should be a single map of named parameters, or a slice of positional parameters.
// It is the caller's responsibility to ensure the correct number and type of parameters.
func (c *Client) QuerySingle(ctx context.Context, statement string, args ...any) (*QueryResponse, error) {
	stmt, err := NewSQLStatement(statement, args...)
	if err != nil {
		return nil, err
	}
	return c.Query(ctx, SQLStatements{stmt}, nil)
}

// Query performs a read operation (SELECT) using /db/query. opts may be nil, in which case default
// options are used.
func (c *Client) Query(ctx context.Context, statements SQLStatements, opts *QueryOptions) (retQr *QueryResponse, retErr error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	queryParams, err := makeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONPostRequest(ctx, queryPath, queryParams, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	}

	var queryResponse QueryResponse
	dec := json.NewDecoder(bytes.NewReader(respBody))
	dec.UseNumber()
	if err := dec.Decode(&queryResponse); err != nil {
		return nil, err
	}
	if c.promoteErrors.Load() {
		if f, i, msg := queryResponse.HasError(); f {
			retErr = fmt.Errorf("statement %d: %s", i, msg)
		}
	}
	return &queryResponse, retErr
}

// RequestSingle sends a single statement, which can be either a read or write.
// args should be a single map of named parameters, or a slice of positional
// parameters. It is the caller's responsibility to ensure the correct number and
// type of parameters.
func (c *Client) RequestSingle(ctx context.Context, statement string, args ...any) (*RequestResponse, error) {
	stmt, err := NewSQLStatement(statement, args...)
	if err != nil {
		return nil, err
	}
	return c.Request(ctx, SQLStatements{stmt}, nil)
}

// Request sends both read and write statements in a single request using /db/request.
// opts may be nil, in which case default options are used.
func (c *Client) Request(ctx context.Context, statements SQLStatements, opts *RequestOptions) (rr *RequestResponse, retErr error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	reqParams, err := makeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONPostRequest(ctx, requestPath, reqParams, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	}

	var reqResp RequestResponse
	dec := json.NewDecoder(bytes.NewReader(respBody))
	dec.UseNumber()
	if err := dec.Decode(&reqResp); err != nil {
		return nil, err
	}
	if c.promoteErrors.Load() {
		if f, i, msg := reqResp.HasError(); f {
			retErr = fmt.Errorf("statement %d: %s", i, msg)
		}
	}
	return &reqResp, retErr
}

// Backup requests a copy of the SQLite database from the node. opts may be nil, in which case
// default options are used. The caller is responsible for closing the returned io.ReadCloser
// when done with it.
func (c *Client) Backup(ctx context.Context, opts *BackupOptions) (rc io.ReadCloser, retError error) {
	defer func() {
		if retError != nil && rc != nil {
			rc.Close()
		}
	}()
	reqParams, err := makeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doGetRequest(ctx, backupPath, reqParams)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// Load streams data from r into the node, to load or restore data. Load automatically
// detects the format of the data, and can handle both plain text and SQLite binary data.
// opts may be nil, in which case default options are used.
func (c *Client) Load(ctx context.Context, r io.Reader, opts *LoadOptions) error {
	params, err := makeURLValues(opts)
	if err != nil {
		return err
	}

	first13 := make([]byte, 13)
	_, err = r.Read(first13)
	if err != nil {
		return err
	}

	if validSQLiteData(first13) {
		_, err = c.doOctetStreamPostRequest(ctx, loadPath, params, io.MultiReader(bytes.NewReader(first13), r))
	} else {
		_, err = c.doPlainPostRequest(ctx, loadPath, params, io.MultiReader(bytes.NewReader(first13), r))
	}
	return err
}

// Boot streams a raw SQLite file into a single-node system, effectively initializing
// the underlying SQLite database from scratch. It is an error to call this on anything
// but a single-node system.
func (c *Client) Boot(ctx context.Context, r io.Reader) error {
	_, err := c.doOctetStreamPostRequest(ctx, bootPath, nil, r)
	return err
}

// RemoveNode removes a node from the cluster. The node is identified by its ID.
func (c *Client) RemoveNode(ctx context.Context, id string) error {
	body := fmt.Sprintf(`{"id":"%s"}`, id)
	resp, err := c.doRequest(ctx, "DELETE", removePath, "application/json", nil, bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	}
	return nil
}

// Status returns the status of the node.
func (c *Client) Status(ctx context.Context) (json.RawMessage, error) {
	resp, err := c.doGetRequest(ctx, statusPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}

// Expvar returns the Go expvar data from the node.
func (c *Client) Expvar(ctx context.Context) (json.RawMessage, error) {
	resp, err := c.doGetRequest(ctx, expvarPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}

// Nodes returns the list of known nodes in the cluster.
func (c *Client) Nodes(ctx context.Context, opts *NodeOptions) (json.RawMessage, error) {
	params, err := makeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doGetRequest(ctx, nodesPath, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}

// Ready returns the readiness of the node.
func (c *Client) Ready(ctx context.Context, opts *ReadyOptions) ([]byte, error) {
	params, err := makeURLValues(opts)
	if err != nil {
		return nil, err
	}
	resp, err := c.doGetRequest(ctx, readyPath, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(b))
	}
	return b, err
}

// Version returns the version of software running on the node.
func (c *Client) Version(ctx context.Context) (string, error) {
	resp, err := c.doGetRequest(ctx, statusPath, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	version := resp.Header.Get("X-RQLITE-VERSION")
	if version == "" {
		version = "unknown"
	}
	return version, nil
}

// Close closes the client and should be called when the client is no longer needed.
func (c *Client) Close() error {
	return nil
}

func (c *Client) doGetRequest(ctx context.Context, path string, values url.Values) (*http.Response, error) {
	return c.doRequest(ctx, "GET", path, "", values, nil)
}

func (c *Client) doJSONPostRequest(ctx context.Context, path string, values url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, "POST", path, "application/json", values, body)
}

func (c *Client) doOctetStreamPostRequest(ctx context.Context, path string, values url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, "POST", path, "application/octet-stream", values, body)
}

func (c *Client) doPlainPostRequest(ctx context.Context, path string, values url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, "POST", path, "text/plain", values, body)
}

// doRequest builds and executes an HTTP request, returning the response.
func (c *Client) doRequest(ctx context.Context, method, path string, contentType string, values url.Values, body io.Reader) (*http.Response, error) {
	baseURL, err := c.lb.Next()
	if err != nil {
		return nil, err
	}
	fullURL := baseURL.JoinPath(path)
	currValues := fullURL.Query()
	maps.Copy(currValues, values)
	fullURL.RawQuery = currValues.Encode()
	c.addUserinfoToURL(fullURL)

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) addUserinfoToURL(u *url.URL) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.basicAuthUser != "" || c.basicAuthPass != "" {
		u.User = url.UserPassword(c.basicAuthUser, c.basicAuthPass)
	}
}

func validSQLiteData(b []byte) bool {
	return len(b) >= 13 && string(b[0:13]) == "SQLite format"
}
