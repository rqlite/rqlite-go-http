package http

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

// BackupOptions holds optional parameters for a backup operation.
type BackupOptions struct {
	// Format can be "sql" if a SQL text dump is desired, otherwise an empty string
	// (or something else) means a binary SQLite file is returned.
	Format string `uvalue:"fmt"`

	// If set, request that the backup be vacuumed before returning it.
	Vacuum bool `uvalue:"vacuum"`

	// If set, request that the backup be GZIP-compressed.
	// e.g. /db/backup?compress
	Compress bool `uvalue:"compress"`

	// If set, ask a Follower not to forward the request to the Leader and instead
	// read its local database and return that as the backup.
	NoLeader bool `uvalue:"noleader"`

	// If set, instruct a Follower to return a redirect instead of forwarding.
	Redirect bool `uvalue:"redirect"`
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
	Timeout time.Duration `uvalue:"timeout"`
}

// QueryOptions holds optional settings for /db/query requests.
type QueryOptions struct {
	// Timeout is applied at the database level.
	Timeout time.Duration `uvalue:"timeout"`

	Pretty  bool `uvalue:"pretty"`
	Timings bool `uvalue:"timings"`

	// Associative signals whether to request the "associative" form of results.
	Associative bool `uvalue:"associative"`

	// BlobAsArray signals whether to request the BLOB data as arrays of byte values.
	BlobAsArray bool `uvalue:"blob_array"`

	Level               string        `uvalue:"level"`
	LinearizableTimeout time.Duration `uvalue:"linearizable_timeout"`
	Freshness           time.Duration `uvalue:"freshness"`
	FreshnessStrict     bool          `uvalue:"freshness_strict"`
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

// MakeURLValues converts a struct to a url.Values, using the `uvalue` tag to
// determine the key name.
func MakeURLValues(input any) (url.Values, error) {
	vals := url.Values{}
	if input == nil {
		return vals, nil
	}

	val := reflect.ValueOf(input)
	typ := reflect.TypeOf(input)

	// If it's a pointer, get the underlying element.
	if typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return vals, nil
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a pointer to a struct, got %s", typ.Kind())
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagVal := field.Tag.Get("uvalue")
		if tagVal == "" {
			// No `uvalue` tag, skip.
			continue
		}

		fieldValue := val.Field(i)
		if !fieldValue.CanInterface() {
			// Unexported or inaccessible field.
			continue
		}

		var strVal string
		if fieldValue.Type() == reflect.TypeOf(time.Duration(0)) {
			d := fieldValue.Interface().(time.Duration)
			strVal = d.String()
		} else {
			switch fieldValue.Kind() {
			case reflect.String:
				strVal = fieldValue.Interface().(string)
			case reflect.Bool:
				b := fieldValue.Interface().(bool)
				strVal = strconv.FormatBool(b)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				i := fieldValue.Int()
				strVal = strconv.FormatInt(i, 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				u := fieldValue.Uint()
				strVal = strconv.FormatUint(u, 10)
			default:
				continue
			}
		}
		vals.Add(tagVal, strVal)
	}
	return vals, nil
}
