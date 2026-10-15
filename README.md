# rqlite-go-http
[![Circle CI](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master)
[![Go Reference](https://pkg.go.dev/badge/github.com/rqlite/rqlite-go-http.svg)](https://pkg.go.dev/github.com/rqlite/rqlite-go-http)

_rqlite-go-http_ is a Go client for rqlite that interacts directly with the rqlite [HTTP API](https://rqlite.io/docs/api/), providing minimal abstraction. Robust and easy-to-use, it can be used on its own or as a foundation for higher-level libraries. The client is safe for concurrent use by multiple goroutines.

This library offers support for:

- Executing SQL statements (`INSERT`, `UPDATE`, `DELETE`)
- Running queries (`SELECT`)
- Handling both read and write statements in a single request via the _Unified Endpoint_.
- Backing up and restoring data to your rqlite system
- Booting a rqlite node from a SQLite database file
- Checking node status, diagnostic info, cluster membership, and readiness
- Ability to customize HTTP communications for control over TLS, mutual TLS, timeouts, etc.

Check out the [documentation](https://pkg.go.dev/github.com/rqlite/rqlite-go-http) for more details.

## Installation

```bash
go get github.com/rqlite/rqlite-go-http
```

### Versioning
This library is under active development and is subject to breaking changes. Be sure to use `go mod` to pin any import of this package to a specific commit.

## Example use

```Go
package main

import (
	"context"
	"fmt"
    "log"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
)

func main() {
	// Create a client pointing to a rqlite node
	client, err := rqlitehttp.NewClient("http://localhost:4001", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Optionally set Basic Auth
	client.SetBasicAuth("user", "password")

	// Create a table.
	resp, err := client.ExecuteSingle(context.Background(), "CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert a record.
	resp, err = client.Execute(
		context.Background(),
		rqlitehttp.SQLStatements{
			{
				SQL:              "INSERT INTO foo(name) VALUES(?)",
				PositionalParams: []any{"fiona"},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ExecuteResponse: %+v\n", resp)

	// Query the newly created table
	qResp, err := client.QuerySingle(context.Background(), "SELECT * FROM foo")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("QueryResponse: %+v\n", qResp)
}
```

## Handling numbers
When handling a JSON response from rqlite which a number, this library stores it as a [`json.Number`](https://pkg.go.dev/encoding/json#Number). This avoids precision loss. You can then convert it to the type your schema expects. For example, if you expect an `int64`:

```go
i, err := row[0].(json.Number).Int64()
if err != nil {
    panic("number is not an int64")
}
```

The JSON specification limits the size of numbers, and the Go standard library JSON encoder represents any number larger than the limit as a string. When working with a JSON response containing such values-as-strings, use the [`math/big`](https://pkg.go.dev/math/big) package:
```go
n := &big.Int{}
v, ok := n.SetString(row[0].(json.Number).String(), 10)
if !ok {
    panic("failed to parse big int")
}
// n now holds the parsed big integer.
```

## Handling errors
A request can fail at two levels. Failures at the HTTP level, such as a node being unreachable or a `5xx` response, are always returned as a Go error. Failures at the database level, such as a constraint violation, are reported inside an HTTP `200` response. Call `Err()` on the response to check for them:

```go
resp, err := client.ExecuteSingle(ctx, "INSERT INTO foo(id) VALUES(1)")
if err != nil {
    log.Fatal(err) // HTTP-level failure.
}
if err := resp.Err(); err != nil {
    log.Fatal(err) // Database-level failure, e.g. "statement 0: UNIQUE constraint failed: foo.id"
}
```

Alternatively, call `client.PromoteErrors(true)` and the client will return the result of `Err()` alongside the response, so a single check covers both levels:

```go
client.PromoteErrors(true)
resp, err := client.ExecuteSingle(ctx, "INSERT INTO foo(id) VALUES(1)")
if err != nil { // Will be true for both HTTP-level errors and database-level failures.
    log.Fatal(err) 
}
```

A failed statement is reported as a `StatementError`, wrapped with the index of the statement that failed. Use `errors.As` to retrieve it. If the failure originated in SQLite, it also carries SQLite's [result codes](https://www.sqlite.org/rescode.html):

```go
var stmtErr *rqlitehttp.StatementError
if errors.As(err, &stmtErr) && stmtErr.SQLite != nil {
    fmt.Println(stmtErr.SQLite.Code, stmtErr.SQLite.ExtendedCode)
}
```

The `Error` field on each result is also set to the `StatementError` for that statement, which is useful when a request contains many statements. The client decodes both the legacy `error` string and the structured `error_v2` object introduced in rqlite v10.6.0, so this works against any version of rqlite.
