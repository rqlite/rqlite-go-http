package rqlitehttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client is the main type through which rqlite is accessed.
type Client struct {
	// httpClient is used for all HTTP requests.
	// A user may override it to customize timeouts, TLS, etc.
	httpClient *http.Client

	executeURL string
	queryURL   string
	requestURL string
	backupURL  string
	loadURL    string
	bootURL    string

	// Fields for optional Basic Auth
	basicAuthUser string
	basicAuthPass string
}

// NewClient creates a new Client with default settings. If httpClient is nil,
// the the default client is used.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	cl := &Client{
		httpClient: httpClient,
		executeURL: baseURL + "/db/execute",
		queryURL:   baseURL + "/db/query",
		requestURL: baseURL + "/db/request",
		backupURL:  baseURL + "/db/backup",
		loadURL:    baseURL + "/db/load",
		bootURL:    baseURL + "/boot",
	}
	if cl.httpClient == nil {
		cl.httpClient = DefaultClient()
	}
	return cl
}

// SetBasicAuth configures the client to use Basic Auth for all subsequent requests.
// Pass empty strings to disable Basic Auth.
func (c *Client) SetBasicAuth(username, password string) {
	c.basicAuthUser = username
	c.basicAuthPass = password
}

// Execute executes one or more SQL statements (INSERT, UPDATE, DELETE) using /db/execute.
func (c *Client) Execute(ctx context.Context, statements SQLStatements, opts ExecuteOptions) (*ExecuteResponse, error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	queryParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", c.executeURL, queryParams, bytes.NewReader(body))
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
	if err := json.Unmarshal(respBody, &executeResp); err != nil {
		return nil, err
	}
	return &executeResp, nil
}

// Query performs a read operation (SELECT) using /db/query.
func (c *Client) Query(ctx context.Context, statements SQLStatements, opts QueryOptions) (*QueryResponse, error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	queryParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", c.queryURL, queryParams, bytes.NewReader(body))
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
	if err := json.Unmarshal(respBody, &queryResponse); err != nil {
		return nil, err
	}
	return &queryResponse, nil
}

// Request sends both read and write statements in a single request using /db/request.
// This method determines read vs. write by inspecting the statements.
func (c *Client) Request(ctx context.Context, statements SQLStatements, opts RequestOptions) (*RequestResponse, error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	reqParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", c.requestURL, reqParams, bytes.NewReader(body))
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
	if err := json.Unmarshal(respBody, &reqResp); err != nil {
		return nil, err
	}
	return &reqResp, nil
}

// Backup requests a copy of the SQLite database from the node. The caller must close the
// returned ReadCloser when done, regardless of any error.
func (c *Client) Backup(ctx context.Context, opts BackupOptions) (io.ReadCloser, error) {
	reqParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "GET", c.backupURL, reqParams, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// Load streams data from r into the node, to load or restore data. Depending on opts.Format,
// the data can be a raw SQLite file (application/octet-stream) or a text dump (text/plain).
//
// This corresponds to a POST request to /db/load.
func (c *Client) Load(ctx context.Context, r io.Reader, opts LoadOptions) error {
	// 1. Build the URL from c.baseURL + "/db/load".
	// 2. Add any query parameters (e.g. redirect if desired).
	// 3. Issue a POST with the appropriate Content-Type:
	//    - application/octet-stream for a raw SQLite file
	//    - text/plain for a SQL dump
	// 4. Upload the data from r.
	// 5. Handle HTTP status codes and parse any JSON error response if needed.

	return nil
}

// Boot streams a raw SQLite file into a single-node system, effectively initializing
// the underlying SQLite store from scratch. This is done via a POST to /boot.
func (c *Client) Boot(ctx context.Context, r io.Reader, opts BootOptions) error {
	// 1. Build the URL from c.baseURL + "/boot".
	// 2. Issue a POST with Transfer-Encoding: chunked or a known Content-Length.
	// 3. The data must be a binary SQLite file. The official doc shows usage:
	//      curl -XPOST 'http://localhost:4001/boot' --upload-file mydb.sqlite
	// 4. Check for HTTP status codes or JSON error messages.

	return nil
}

// Close can clean up any long-lived resources owned by the Client, if needed.
func (c *Client) Close() error {
	return nil
}

// ExecuteResponse represents the JSON returned by /db/execute.
type ExecuteResponse struct {
	Results        []ExecuteResult `json:"results"`
	Time           float64         `json:"time"`
	SequenceNumber int64           `json:"sequence_number,omitempty"`
}

// ExecuteResult is an element of ExecuteResponse.Results.
type ExecuteResult struct {
	LastInsertID *int64  `json:"last_insert_id,omitempty"`
	RowsAffected *int64  `json:"rows_affected,omitempty"`
	Time         float64 `json:"time,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// QueryResponse represents the JSON returned by /db/query in the default (non-associative) form.
type QueryResponse struct {
	Results []QueryResult `json:"results"`
	Time    float64       `json:"time"`
}

// QueryResult is an element of QueryResponse.Results.
type QueryResult struct {
	Columns []string `json:"columns,omitempty"`
	Types   []string `json:"types,omitempty"`
	Values  [][]any  `json:"values,omitempty"`
	Time    float64  `json:"time,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// RequestResponse represents the JSON returned by /db/request.
type RequestResponse struct {
	Results []RequestResult `json:"results"`
	Time    float64         `json:"time"`
}

// RequestResult is an element of RequestResponse.Results.
// It may include either Query-like results or Execute-like results, or an error.
type RequestResult struct {
	// Same fields as QueryResult plus ExecuteResult fields.
	// If read-only, LastInsertID and RowsAffected would be empty;
	// if write-only, Columns and Values would be empty.
	Columns      []string `json:"columns,omitempty"`
	Types        []string `json:"types,omitempty"`
	Values       [][]any  `json:"values,omitempty"`
	LastInsertID *int64   `json:"last_insert_id,omitempty"`
	RowsAffected *int64   `json:"rows_affected,omitempty"`
	Error        string   `json:"error,omitempty"`
	Time         float64  `json:"time,omitempty"`
	// If associative form is requested, you could define a special type for that case,
	// or include an alternative representation of rows here.
}

// doRequest builds and executes an HTTP request, returning the response.
func (c *Client) doRequest(ctx context.Context, method, url string, values url.Values, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url+"?"+values.Encode(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// If Basic Auth is configured, add an Authorization header
	if c.basicAuthUser != "" || c.basicAuthPass != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.basicAuthUser, c.basicAuthPass)))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
