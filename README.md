# rqlite-go-http
[![Circle CI](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/rqlite/rqlite)](https://goreportcard.com/report/github.com/rqlite/rqlite-go-http)
[![Go Reference](https://pkg.go.dev/badge/github.com/rqlite/rqlite-go-http.svg)](https://pkg.go.dev/github.com/rqlite/rqlite-go-http)

A "thin" Go-based client for [rqlite](https://rqlite.io) that communicates with its [HTTP API](https://rqlite.io/docs/api/). This client is useful on its own or as a foundation for higher-level libraries.

This library offers support for:

- Executing SQL statements (`INSERT`, `UPDATE`, `DELETE`)
- Running queries (`SELECT`)
- Handling both read and write statements in a single request via the _Unified Endpoint_.
- Backing up and restoring data to your rqlite system
- Booting a rqlite node from a SQLite database file
- Checking node status, diagnostic info, cluster membership, and readiness
- Ability to customize HTTP communications for control over TLS, mutual TLS, timeouts, etc.

## Installation

```bash
go get github.com/rqlite/http
```

## Example use

```Go
package main

import (
	"context"
	"fmt"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
)

func main() {
	// Create a client pointing to a rqlite node
	client := rqlitehttp.NewClient("http://localhost:4001", nil)

	// Optionally set Basic Auth
	client.SetBasicAuth("user", "password")

	// Create a table.
	resp, err := client.ExecuteSingle(context.Background(), "CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		panic(err)
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
		panic(err)
	}
	fmt.Printf("ExecuteResponse: %+v\n", resp)

	// Query the newly created table
	qResp, err := client.QuerySingle(context.Background(), "SELECT * FROM foo")
	if err != nil {
		panic(err)
	}
	fmt.Printf("QueryResponse: %+v\n", qResp)
}
```
