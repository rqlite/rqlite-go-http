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

// Close can clean up any long-lived resources owned by the Client, if needed.
func (c *Client) Close() error {
    return nil
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
}

// ExecuteResponse represents the JSON returned by /db/execute.
type ExecuteResponse struct {
    Results []ExecuteResult `json:"results"`
    Time    float64         `json:"time"`
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
