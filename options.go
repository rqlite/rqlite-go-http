package http

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ReadConsistencyLevel int

const (
	ReadConsistencyLevelUnknown = iota
	ReadConsistencyLevelNone
	ReadConsistencyLevelWeak
	ReadConsistencyLevelStrong
	ReadConsistencyLevelLinearizable
	ReadConsistencyLevelAuto
)

func (rcl ReadConsistencyLevel) String() string {
	switch rcl {
	case ReadConsistencyLevelNone:
		return "none"
	case ReadConsistencyLevelWeak:
		return "weak"
	case ReadConsistencyLevelStrong:
		return "strong"
	case ReadConsistencyLevelLinearizable:
		return "linearizable"
	case ReadConsistencyLevelAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// BackupOptions holds optional parameters for a backup operation.
type BackupOptions struct {
	// Format can be "sql" if a SQL text dump is desired, otherwise an empty string
	// (or something else) means a binary SQLite file is returned.
	Format string `uvalue:"fmt,omitempty"`

	// If set, request that the backup be vacuumed before returning it.
	Vacuum bool `uvalue:"vacuum,omitempty"`

	// If set, request that the backup be GZIP-compressed.
	// e.g. /db/backup?compress
	Compress bool `uvalue:"compress,omitempty"`

	// If set, ask a Follower not to forward the request to the Leader and instead
	// read its local database and return that as the backup.
	NoLeader bool `uvalue:"noleader,omitempty"`

	// If set, instruct a Follower to return a redirect instead of forwarding.
	Redirect bool `uvalue:"redirect,omitempty"`
}

// LoadOptions configures how to load data into the node.
type LoadOptions struct {
	// If set, instruct a Follower to return a redirect instead of forwarding.
	// e.g. /db/load?redirect
	Redirect bool `uvalue:"redirect,omitempty"`
}

// ExecuteOptions holds optional settings for /db/execute requests.
type ExecuteOptions struct {
	// Transaction indicates whether statements should be enclosed in a transaction.
	Transaction bool `uvalue:"transaction,omitempty"`

	// Pretty requests pretty-printed JSON.
	Pretty bool `uvalue:"pretty,omitempty"`

	// Timings requests timing information.
	Timings bool `uvalue:"timings,omitempty"`

	// Queue requests that the statement be queued
	Queue bool `uvalue:"queue,omitempty"`

	// Wait requests that the system only respond once the statement has been committed.
	// This is ignored unless Queue is true. If Queue is not true, an Execute request
	// always waits until the request has been committed.
	Wait bool `uvalue:"wait,omitempty"`

	// Timeout after which if Wait is set, the system should respond with an error if
	// the request has not been persisted.
	Timeout time.Duration `uvalue:"timeout,omitempty"`
}

// QueryOptions holds optional settings for /db/query requests.
type QueryOptions struct {
	// Timeout is applied at the database level.
	Timeout time.Duration `uvalue:"timeout,omitempty"`

	Pretty  bool `uvalue:"pretty,omitempty"`
	Timings bool `uvalue:"timings,omitempty"`

	// Associative signals whether to request the "associative" form of results.
	Associative bool `uvalue:"associative,omitempty"`

	// BlobAsArray signals whether to request the BLOB data as arrays of byte values.
	BlobAsArray bool `uvalue:"blob_array,omitempty"`

	Level               ReadConsistencyLevel `uvalue:"level,omitempty"`
	LinearizableTimeout time.Duration        `uvalue:"linearizable_timeout,omitempty"`
	Freshness           time.Duration        `uvalue:"freshness,omitempty"`
	FreshnessStrict     bool                 `uvalue:"freshness_strict,omitempty"`
}

// RequestOptions holds optional settings for /db/request requests.
type RequestOptions struct {
	// Transaction indicates whether statements should be enclosed in a transaction.
	Transaction bool `uvalue:"transaction,omitempty"`

	// Timeout is applied at the database level.
	Timeout     time.Duration `uvalue:"timeout,omitempty"`
	Pretty      bool          `uvalue:"pretty,omitempty"`
	Timings     bool          `uvalue:"timings,omitempty"`
	Associative bool          `uvalue:"associative,omitempty"`
	BlobAsArray bool          `uvalue:"blob_array,omitempty"`

	Level               ReadConsistencyLevel `uvalue:"level,omitempty"`
	LinearizableTimeout string               `uvalue:"linearizable_timeout,omitempty"`
	Freshness           string               `uvalue:"freshness,omitempty"`
	FreshnessStrict     bool                 `uvalue:"freshness_strict,omitempty"`
}

// NodeOptions holds optional settings for /nodes requests.
type NodeOptions struct {
	Timeout   time.Duration `uvalue:"timeout,omitempty"`
	Pretty    bool          `uvalue:"pretty,omitempty"`
	NonVoters bool          `uvalue:"non_voters,omitempty"`
	Version   string        `uvalue:"ver,omitempty"`
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
		parts := strings.Split(tagVal, ",")
		tagVal = parts[0]
		omitEmpty := false
		if len(parts) > 1 {
			// If there are multiple parts, the second part is the option.
			omitEmpty = parts[1] == "omitempty"
		}

		fieldValue := val.Field(i)
		if !fieldValue.CanInterface() {
			// Unexported or inaccessible field.
			continue
		}

		var strVal string
		if fieldValue.Type() == reflect.TypeOf(time.Duration(0)) {
			d := fieldValue.Interface().(time.Duration)
			if d == 0 && omitEmpty {
				continue
			}
			strVal = d.String()
		} else if fieldValue.Type() == reflect.TypeOf(ReadConsistencyLevel(0)) {
			rcl := fieldValue.Interface().(ReadConsistencyLevel)
			if rcl == ReadConsistencyLevelUnknown {
				continue
			}
			strVal = rcl.String()
		} else {
			switch fieldValue.Kind() {
			case reflect.String:
				strVal = fieldValue.Interface().(string)
				if omitEmpty && strVal == "" {
					continue
				}
			case reflect.Bool:
				b := fieldValue.Interface().(bool)
				if omitEmpty && !b {
					continue
				}
				strVal = strconv.FormatBool(b)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				i := fieldValue.Int()
				if omitEmpty && i == 0 {
					continue
				}
				strVal = strconv.FormatInt(i, 10)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				u := fieldValue.Uint()
				if omitEmpty && u == 0 {
					continue
				}
				strVal = strconv.FormatUint(u, 10)
			default:
				continue
			}
		}
		vals.Add(tagVal, strVal)
	}
	return vals, nil
}
