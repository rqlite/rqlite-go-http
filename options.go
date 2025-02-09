package rqlitehttp

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

// ExecuteOptions holds optional settings for /db/execute requests.
type ExecuteOptions struct {
	// Transaction indicates whether statements should be enclosed in a transaction.
	Transaction bool `uvalue:"transaction"`

	// Pretty requests pretty-printed JSON.
	Pretty bool `uvalue:"pretty"`

	// Timings requests timing information.
	Timings bool `uvalue:"timings"`

	// Queue requests that the statement be queued
	Queue bool `uvalue:"queue"`

	// Wait requests that the system only respond once the statement has been committed.
	Wait bool `uvalue:"wait"`

	// Timeout after which if Wait is set, the system should respond with an error if
	// the request has not been persisted.
	Timeout string `uvalue:"timeout"`
}

// QueryOptions holds optional settings for /db/query requests.
type QueryOptions struct {
	// Timeout is applied at the database level.
	Timeout string

	Pretty  bool
	Timings bool

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
