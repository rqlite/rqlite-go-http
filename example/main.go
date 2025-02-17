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
