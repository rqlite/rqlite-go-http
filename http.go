package rqlitehttp

import (
    "context"
    "net/http"
    "time"
)

// Client is the main type through which rqlite is accessed.
type Client struct {
    // httpClient is used for all HTTP requests.
    // A user may override it to customize timeouts, TLS, etc.
    httpClient *http.Client

    // baseURL is the HTTP address of a single rqlite node (for example "http://localhost:4001").
    // You may implement logic to discover or switch nodes in case of network errors or cluster changes.
    baseURL string

    // defaultParams are parameters like `pretty`, `timings`, or custom headers that apply to all requests.
    // You may override or add to them on a per-request basis.
    defaultParams map[string]string

    // Fields for optional Basic Auth
    basicAuthUser string
    basicAuthPass string
}

// NewClient creates a new Client with default settings.
func NewClient(baseURL string, httpClient *http.Client) *Client {
    if httpClient == nil {
        httpClient = &http.Client{Timeout: 10 * time.Second}
    }

    return &Client{
        baseURL:     baseURL,
        httpClient:  httpClient,
        defaultParams: make(map[string]string),
    }
}

// SetBasicAuth configures the client to use Basic Auth for all subsequent requests.
// Pass empty strings to disable Basic Auth.
func (c *Client) SetBasicAuth(username, password string) {
    c.basicAuthUser = username
    c.basicAuthPass = password
}

// Execute executes one or more SQL statements (INSERT, UPDATE, DELETE, or DDL) using /db/execute.
// The statements parameter is a slice of SQL statements, which can contain parameter placeholders if needed.
func (c *Client) Execute(ctx context.Context, statements []SQLStatement, opts ExecuteOptions) (*ExecuteResponse, error) {
    // Implementation to be added
    return nil, nil
}

// Query performs a read operation (SELECT) using /db/query. The statements can be passed as
// SQL strings or parameterized statements. The user may set query parameters in opts.
func (c *Client) Query(ctx context.Context, statements []SQLStatement, opts QueryOptions) (*QueryResponse, error) {
    // Implementation to be added
    return nil, nil
}

// Request sends both read and write statements in a single request using /db/request.
// This method determines read vs. write by inspecting the statements.
func (c *Client) Request(ctx context.Context, statements []SQLStatement, opts RequestOptions) (*RequestResponse, error) {
    // Implementation to be added
    return nil, nil
}


// -------------------------------------------------------------
// Backup
// -------------------------------------------------------------

// Backup requests a copy of the SQLite database (or a SQL text dump) from the node.
// The returned data can be saved to disk or processed in memory.
//
// This method issues a GET request to /db/backup. If opts.Fmt == "sql", it requests
// a SQL text dump instead of a binary SQLite file.
func (c *Client) Backup(ctx context.Context, opts BackupOptions) (io.ReadCloser, error) {
    // 1. Build the URL from c.baseURL + "/db/backup".
    // 2. Add query parameters as indicated by opts.
    // 3. Create an HTTP GET request.
    // 4. Return the response body (io.ReadCloser) if successful.

    return nil, nil
}

// BackupOptions holds optional parameters for a backup operation.
type BackupOptions struct {
    // Fmt can be "sql" if a SQL text dump is desired, otherwise an empty string
    // (or something else) means a binary SQLite file is returned.
    Fmt string

    // If set, request that the backup be vacuumed before returning it.
    // e.g. /db/backup?vacuum
    Vacuum bool

    // If set, request that the backup be GZIP-compressed.
    // e.g. /db/backup?compress
    Compress bool

    // If set, ask a Follower not to forward the request to the Leader.
    // e.g. /db/backup?noleader
    NoLeader bool

    // If set, instruct a Follower to return a redirect instead of forwarding.
    // e.g. /db/backup?redirect
    Redirect bool
}

// -------------------------------------------------------------
// Restore (Load or Boot)
// -------------------------------------------------------------

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

// LoadOptions configures how to load data into the node.
type LoadOptions struct {
    // Format can be "binary" or "sql" etc.
    // - "binary" -> application/octet-stream
    // - "sql"    -> text/plain
    Format string

    // If set, instruct a Follower to return a redirect instead of forwarding.
    // e.g. /db/load?redirect
    Redirect bool
}

// BootOptions configures how to boot a single-node system.
type BootOptions struct {
    // Potential expansions (for instance, forcing a redirect or not).
    // Usually /boot is only relevant for a single-node system, so
    // there's not too much to configure.
}

// SQLStatement represents a single SQL statement, possibly with parameters.
type SQLStatement struct {
    // SQL is the text of the SQL statement, for example "INSERT INTO foo VALUES(?)".
    SQL string

    // PositionalParams is a slice of values for placeholders (?), if used.
    PositionalParams []interface{}

    // NamedParams is a map of parameter names to values, if using named placeholders.
    NamedParams map[string]interface{}
}

// ExecuteOptions holds optional settings for /db/execute requests.
type ExecuteOptions struct {
    // Transaction indicates whether statements should be enclosed in a transaction.
    Transaction bool

    // Timeout is applied at the database level, for example "2s".
    // Internally this might translate to the `db_timeout` query parameter.
    Timeout string

    // Additional query parameters like "pretty" or "timings" can be added here as needed.
    Pretty  bool
    Timings bool

    Queue       bool   // If true, add ?queue
    Wait        bool   // If true, also add &wait
    WaitTimeout string // If non-empty, add &timeout=<value>, e.g. "10s"
}

// QueryOptions holds optional settings for /db/query requests.
type QueryOptions struct {
    // Timeout is applied at the database level.
    Timeout string

    Pretty      bool
    Timings     bool

    // Associative signals whether to request the "associative" form of results.
    Associative bool

    // BlobAsArray signals whether to request the BLOB data as arrays of byte values.
    BlobAsArray bool

    Level               string // "weak" (default), "linearizable", "strong", "none", or "auto".
    LinearizableTimeout string // e.g. "1s" if level=linearizable.
    Freshness           string // e.g. "1s" if level=none.
    FreshnessStrict     bool   // if true, adds &freshness_strict.
}

// RequestOptions holds optional settings for /db/request requests.
type RequestOptions struct {
    // Transaction indicates whether statements should be enclosed in a transaction.
    Transaction bool

    // Timeout is applied at the database level.
    Timeout     string
    Pretty      bool
    Timings     bool
    Associative bool
    BlobAsArray bool

    Level               string // "weak" (default), "linearizable", "strong", "none", or "auto".
    LinearizableTimeout string // e.g. "1s" if level=linearizable.
    Freshness           string // e.g. "1s" if level=none.
    FreshnessStrict     bool   // if true, adds &freshness_strict.
}

// ExecuteResponse represents the JSON returned by /db/execute.
type ExecuteResponse struct {
    Results []ExecuteResult `json:"results"`
    Time    float64         `json:"time"`
    SequenceNumber int64   `json:"sequence_number,omitempty"`
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
    Columns []string        `json:"columns,omitempty"`
    Types   []string        `json:"types,omitempty"`
    Values  [][]interface{} `json:"values,omitempty"`
    Time    float64         `json:"time,omitempty"`
    Error   string          `json:"error,omitempty"`
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
    Columns       []string        `json:"columns,omitempty"`
    Types         []string        `json:"types,omitempty"`
    Values        [][]interface{} `json:"values,omitempty"`
    LastInsertID  *int64          `json:"last_insert_id,omitempty"`
    RowsAffected  *int64          `json:"rows_affected,omitempty"`
    Error         string          `json:"error,omitempty"`
    Time          float64         `json:"time,omitempty"`
    // If associative form is requested, you could define a special type for that case,
    // or include an alternative representation of rows here.
}

// addQueryParams takes a base URL (like "http://localhost:4001/db/query")
// and a map of query key-value pairs. It returns the complete URL with
// encoded parameters, for example:
//   "http://localhost:4001/db/query?level=weak&pretty&timings"
func (c *Client) addQueryParams(base string, params map[string]string) (string, error) {
    // 1. Parse baseURL
    // 2. Add query params from the map
    // 3. Return combined URL
    return "", nil
}

// doRequest builds and executes an HTTP request, returning the response.
// This can handle setting Content-Type, attaching the context, etc.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, method, url, body)
    if err != nil {
        return nil, err
    }

    // Add any passed-in headers
    for k, v := range headers {
        req.Header.Set(k, v)
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

// decodeJSONResponse reads and unmarshals JSON from r into dest.
// This can be used by Query, Execute, etc., to parse responses consistently.
func decodeJSONResponse(r io.Reader, dest interface{}) error {
    // 1. Read all or use JSON decoder
    // 2. Unmarshal into dest
    // 3. Return any errors
    return nil
}

// marshalStatementsToJSON is one place to handle the "SQL statements array" format
// required by rqlite for both /db/query and /db/execute, rather than duplicating
// that logic in multiple places. 
func marshalStatementsToJSON(statements []SQLStatement) ([]byte, error) {
    // 1. Build an array of interface{} representing each statement
    //    If a statement has PositionalParams or NamedParams, represent
    //    it as [SQL, param1, param2] or [SQL, { "name": "foo", "value": 5 }] etc.
    // 2. json.Marshal(...)
    return nil, nil
}

