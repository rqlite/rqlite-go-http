# rqlite-go-http
[![Circle CI](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master)

A "thin" Go-based client for [rqlite](https://github.com/rqlite/rqlite) that communicates with its HTTP interface. This client is useful on its own or as a foundation for higher-level libraries.

This library offers endpoints for:

- Executing SQL statements (`INSERT`, `UPDATE`, `DELETE`)
- Running queries (`SELECT`)
- Handling both read/write statements in a single request
- Backing up and restoring data
- Booting a node from a raw SQLite file
- Checking node status, diagnostic info, cluster membership, and readiness

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
				PositionalParams: []interface{}{"fiona"},
			},
		},
		nil, // optional ExecuteOptions
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
