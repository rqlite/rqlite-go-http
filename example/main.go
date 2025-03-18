package main

import (
	"context"
	"encoding/json"
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
	if f, _, err := resp.HasError(); f {
		panic(err)
	}

	// Insert a record.
	resp, err = client.ExecuteSingle(context.Background(), "INSERT INTO foo(name) VALUES(?)", "fiona")
	if err != nil {
		panic(err)
	}
	if f, _, err := resp.HasError(); f {
		panic(err)
	}

	// Insert a second record with full control.
	resp, err = client.Execute(
		context.Background(),
		rqlitehttp.SQLStatements{
			{
				SQL:              "INSERT INTO foo(name) VALUES(?)",
				PositionalParams: []any{"declan"},
			},
		},
		&rqlitehttp.ExecuteOptions{
			Timings: true,
		},
	)
	if err != nil {
		panic(err)
	}
	if f, _, err := resp.HasError(); f {
		panic(err)
	}
	fmt.Printf("ExecuteResponse: %s\n", jsonMarshal(resp))

	// Query the newly created table
	qResp, err := client.QuerySingle(context.Background(), "SELECT * FROM foo")
	if err != nil {
		panic(err)
	}
	if f, _, err := resp.HasError(); f {
		panic(err)
	}
	fmt.Printf("QueryResponse: %s\n", jsonMarshal(qResp))
}

func jsonMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
