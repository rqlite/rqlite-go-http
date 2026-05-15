# rqlite-go-http
[![Circle CI](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/rqlite-go-http/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/rqlite/rqlite)](https://goreportcard.com/report/github.com/rqlite/rqlite-go-http)
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

## database/sql driver

The `stdlib/rqlite` package registers rqlite as a [`database/sql`](https://pkg.go.dev/database/sql) driver, allowing you to use the standard Go database interface.

This driver supports buffered write-only transactions.
Running a read query where rows are expected to be returned during a transaction will immediately fail with an error.
Buffering means all queries will be executed on `Commit()` in a single batch and all errors will be returned by it as a single error value.
Rolling back transactions is effectively no-op as nothing will be executed until `Commit()` is called.

Said limitation can be relaxed by enabling `AllowQueryInTxn` driver flag but these will be executed out-of-band outside the transaction not reflecting any state that would be modified by it.

For extended configuration, import `stdlib` directly and register your own `stdlib.Driver`.

```Go
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/rqlite/rqlite-go-http/stdlib/rqlite"
)

func main() {
	db, err := sql.Open("rqlite", "http://localhost:4001")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec("INSERT INTO foo(name) VALUES(?)", "fiona"); err != nil {
		log.Fatal(err)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM foo WHERE id = 1").Scan(&name); err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	rows, err := db.Query("SELECT id, name FROM foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d: %s\n", id, n)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
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
