package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
)

// ExecuteResponse represents the JSON returned by /db/execute.
type ExecuteResponse struct {
	Results        []ExecuteResult `json:"results"`
	Time           float64         `json:"time,omitempty"`
	SequenceNumber int64           `json:"sequence_number,omitempty"`
}

// HasError returns true if any of the results in the response contain an error.
// If an error is found, the index of the result and the error message are returned.
func (er *ExecuteResponse) HasError() (bool, int, string) {
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
	Results any     `json:"results"`
	Time    float64 `json:"time,omitempty"`
}

// HasError returns true if any of the results in the response contain an error.
// If an error is found, the index of the result and the error message are returned.
func (qr *QueryResponse) HasError() (bool, int, string) {
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
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Attempt to unmarshal Results into []QueryResult.
	var res []QueryResult
	if err := json.Unmarshal(aux.Results, &res); err == nil {
		qr.Results = res
		return nil
	}

	// Attempt to unmarshal Results into []QueryResultAssoc.
	var resAssoc []QueryResultAssoc
	if err := json.Unmarshal(aux.Results, &resAssoc); err == nil {
		qr.Results = resAssoc
		return nil
	}

	return fmt.Errorf("unable to unmarshal results into either []QueryResult or []QueryResultAssoc")
}

// QueryResult is an element of QueryResponse.Results.
type QueryResult struct {
	Columns []string `json:"columns"`
	Types   []string `json:"types"`
	Values  [][]any  `json:"values"`
	Time    float64  `json:"time,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type QueryResultAssoc struct {
	Types map[string]string `json:"types"`
	Rows  []map[string]any  `json:"rows"`
	Time  float64           `json:"time,omitempty"`
	Error string            `json:"error,omitempty"`
}

// RequestResponse represents the JSON returned by /db/request.
type RequestResponse struct {
	Results []RequestResult `json:"results"`
	Time    float64         `json:"time,omitempty"`
}

// HasError returns true if any of the results in the response contain an error.
// If an error is found, the index of the result and the error message are returned.
func (rr *RequestResponse) HasError() (bool, int, string) {
	for i, result := range rr.Results {
		if result.Error != "" {
			return true, i, result.Error
		}
	}
	return false, -1, ""
}

// RequestResult is an element of RequestResponse.Results.
// It may include either Query-like results or Execute-like results, or an error.
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

// Client is the main type through which rqlite is accessed.
type Client struct {
	httpClient *http.Client

	executeURL string
	queryURL   string
	requestURL string
	backupURL  string
	loadURL    string
	bootURL    string
	statusURL  string
	expvarURL  string
	nodesURL   string
	readyURL   string

	basicAuthUser string
	basicAuthPass string

	promoteErrors atomic.Bool
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
		statusURL:  baseURL + "/status",
		expvarURL:  baseURL + "/debug/vars",
		nodesURL:   baseURL + "/nodes",
		readyURL:   baseURL + "/readyz",
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

// PromoteErrors enables or disables the promotion of statement-level errors to Go errors.
//
// By default an operation on the client only returns an error if there is a failure at
// the HTTP level. If this method is called with true, then the client will also return
// an error if there is any failure at the statement level, setting the returned error
// to the first statement-level error encountered.
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
func (c *Client) Execute(ctx context.Context, statements SQLStatements, opts *ExecuteOptions) (retEr *ExecuteResponse, retErr error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	queryParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONPostRequest(ctx, c.executeURL, queryParams, bytes.NewReader(body))
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

// Query performs a read operation (SELECT) using /db/query.
func (c *Client) Query(ctx context.Context, statements SQLStatements, opts *QueryOptions) (retQr *QueryResponse, retErr error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	queryParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONPostRequest(ctx, c.queryURL, queryParams, bytes.NewReader(body))
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
	if c.promoteErrors.Load() {
		if f, i, msg := queryResponse.HasError(); f {
			retErr = fmt.Errorf("statement %d: %s", i, msg)
		}
	}
	return &queryResponse, retErr
}

// RequestSingle sends a single statement using /db/request. args should be a single map
// of named parameters, or a slice of positional parameters.
// It is the caller's responsibility to ensure the correct number and type of parameters.
func (c *Client) RequestSingle(ctx context.Context, statement string, args ...any) (*RequestResponse, error) {
	stmt, err := NewSQLStatement(statement, args...)
	if err != nil {
		return nil, err
	}
	return c.Request(ctx, SQLStatements{stmt}, nil)
}

// Request sends both read and write statements in a single request using /db/request.
// This method determines read vs. write by inspecting the statements.
func (c *Client) Request(ctx context.Context, statements SQLStatements, opts *RequestOptions) (rr *RequestResponse, retErr error) {
	body, err := statements.MarshalJSON()
	if err != nil {
		return nil, err
	}
	reqParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doJSONPostRequest(ctx, c.requestURL, reqParams, bytes.NewReader(body))
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
	if c.promoteErrors.Load() {
		if f, i, msg := reqResp.HasError(); f {
			retErr = fmt.Errorf("statement %d: %s", i, msg)
		}
	}
	return &reqResp, retErr
}

// Backup requests a copy of the SQLite database from the node. The caller must close the
// returned ReadCloser when done, regardless of any error.
func (c *Client) Backup(ctx context.Context, opts BackupOptions) (io.ReadCloser, error) {
	reqParams, err := MakeURLValues(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.doGetRequest(ctx, c.backupURL, reqParams)
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
func (c *Client) Load(ctx context.Context, r io.Reader, opts LoadOptions) error {
	params, err := MakeURLValues(opts)
	if err != nil {
		return err
	}
	_ = params

	first13 := make([]byte, 13)
	_, err = r.Read(first13)
	if err != nil {
		return err
	}

	if validSQLiteData(first13) {
		_, err = c.doOctetStreamPostRequest(ctx, c.loadURL, params, io.MultiReader(bytes.NewReader(first13), r))
	} else {
		_, err = c.doPlainPostRequest(ctx, c.loadURL, params, io.MultiReader(bytes.NewReader(first13), r))
	}
	return err
}

// Boot streams a raw SQLite file into a single-node system, effectively initializing
// the underlying SQLite store from scratch. This is done via a POST to /boot.
func (c *Client) Boot(ctx context.Context, r io.Reader) error {
	_, err := c.doOctetStreamPostRequest(ctx, c.bootURL, nil, r)
	return err
}

// Status returns the status of the node.
func (c *Client) Status(ctx context.Context) (json.RawMessage, error) {
	resp, err := c.doGetRequest(ctx, c.statusURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// Expvar returns the expvar data from the node.
func (c *Client) Expvar(ctx context.Context) (json.RawMessage, error) {
	resp, err := c.doGetRequest(ctx, c.expvarURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// Nodes returns the list of known nodes in the cluster.
func (c *Client) Nodes(ctx context.Context) (json.RawMessage, error) {
	resp, err := c.doGetRequest(ctx, c.nodesURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// Ready returns the readiness of the node. The caller must close the returned ReadCloser
// when done, regardless of any error.
func (c *Client) Ready(ctx context.Context) (io.ReadCloser, error) {
	resp, err := c.doGetRequest(ctx, c.readyURL, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Close can clean up any long-lived resources owned by the Client, if needed.
func (c *Client) Close() error {
	return nil
}

func (c *Client) doGetRequest(ctx context.Context, url string, values url.Values) (*http.Response, error) {
	return c.doRequest(ctx, "GET", url, "", values, nil)
}

func (c *Client) doJSONPostRequest(ctx context.Context, url string, values url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, "POST", url, "application/json", values, body)
}

func (c *Client) doOctetStreamPostRequest(ctx context.Context, url string, values url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, "POST", url, "application/octet-stream", values, body)
}

func (c *Client) doPlainPostRequest(ctx context.Context, url string, values url.Values, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, "POST", url, "text/plain", values, body)
}

// doRequest builds and executes an HTTP request, returning the response.
func (c *Client) doRequest(ctx context.Context, method, url string, contentTpe string, values url.Values, body io.Reader) (*http.Response, error) {
	fullURL := url
	if values != nil {
		fullURL += "?" + values.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}
	if contentTpe != "" {
		req.Header.Set("Content-Type", contentTpe)
	}

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

func validSQLiteData(b []byte) bool {
	return len(b) > 13 && string(b[0:13]) == "SQLite format"
}
