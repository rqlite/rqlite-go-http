package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
)

// checkStatementError verifies that err wraps a StatementError carrying the
// SQLite codes for SQLITE_ERROR, as rqlite reports for a duplicate table.
func checkStatementError(t *testing.T, err error) {
	t.Helper()
	var stmtErr *StatementError
	if !errors.As(err, &stmtErr) {
		t.Fatalf("Expected error to wrap a StatementError, got %T: %v", err, err)
	}
	if stmtErr.SQLite == nil {
		t.Fatalf("Expected SQLite error codes, got none: %v", err)
	}
	if stmtErr.SQLite.Code != 1 || stmtErr.SQLite.ExtendedCode != 1 {
		t.Fatalf("Unexpected SQLite error codes: %+v", stmtErr.SQLite)
	}
}

func Test_EndToEnd(t *testing.T) {
	host, ok := os.LookupEnv("RQLITE_GO_HTTP_E2E_HOST")
	if !ok {
		t.Skip("Skipping end-to-end test since no host is set")
	}
	ctx := context.Background()

	client, err := NewClient(fmt.Sprintf("http://%s:4001", host), nil)
	if err != nil {
		t.Fatalf("Error creating client: %s", err)
	}
	defer client.Close()

	if _, err := client.ExecuteSingle(ctx, "DROP TABLE IF EXISTS foo"); err != nil {
		t.Fatalf("Error creating table: %s", err)
	}

	if _, err := client.ExecuteSingle(ctx, "CREATE TABLE foo (id INT, name TEXT)"); err != nil {
		t.Fatalf("Error creating table: %s", err)
	}

	if _, err := client.ExecuteSingle(ctx, "CREATE TABLE foo (id INT, name TEXT)"); err != nil {
		t.Fatalf("Unexpected error creating an already created table: %s", err)
	}
	client.PromoteErrors(true)
	_, err = client.ExecuteSingle(ctx, "CREATE TABLE foo (id INT, name TEXT)")
	if err == nil {
		t.Fatalf("Expected error creating table duplicate table")
	}
	checkStatementError(t, err)

	// Whichever error format is requested, the client must surface the same error.
	for _, format := range []string{ErrorFormatV1, ErrorFormatV2} {
		stmts := NewSQLStatementsFromStrings([]string{"CREATE TABLE foo (id INT, name TEXT)"})
		_, err = client.Execute(ctx, stmts, &ExecuteOptions{Errors: format})
		if err == nil {
			t.Fatalf("Expected error creating duplicate table with errors=%s", format)
		}
		if format == ErrorFormatV2 {
			checkStatementError(t, err)
		}
	}

	stmt, err := NewSQLStatement("INSERT INTO foo(name) VALUES(?)", "fiona")
	if err != nil {
		t.Fatalf("Error creating statement: %s", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := client.Execute(ctx, SQLStatements{stmt}, nil); err != nil {
			t.Fatalf("Error inserting record %d: %s", i, err)
		}
	}

	stmt, err = NewSQLStatement("SELECT COUNT(*) FROM foo")
	if err != nil {
		t.Fatalf("Error creating statement: %s", err)
	}
	resp, err := client.Query(ctx, SQLStatements{stmt}, nil)
	if err != nil {
		t.Fatalf("Error counting records: %s", err)
	}

	results := resp.GetQueryResults()
	if len(results) != 1 {
		t.Fatalf("Unexpected number of results")
	}
	if len(results[0].Values) != 1 {
		t.Fatalf("Unexpected number of rows")
	}
	v, ok := results[0].Values[0][0].(json.Number)
	if !ok {
		t.Fatalf("Unexpected value type: %T", results[0].Values[0][0])
	}
	if v.String() != "10" {
		t.Fatalf("Unexpected value")
	}

	reqResp, err := client.Request(ctx, NewSQLStatementsFromStrings([]string{
		`INSERT INTO foo(name) VALUES("fiona")`, `SELECT COUNT(*) FROM foo`}), nil)
	if err != nil {
		t.Fatalf("Error counting records: %s", err)
	}
	reqResults := reqResp.GetRequestResults()
	if len(reqResults) != 2 {
		t.Fatalf("Unexpected number of results")
	}
	if len(reqResults[1].Values) != 1 {
		t.Fatalf("Unexpected number of rows")
	}
	v, ok = reqResults[1].Values[0][0].(json.Number)
	if !ok {
		t.Fatalf("Unexpected value type: %T", reqResults[0].Values[0][0])
	}
	if v.String() != "11" {
		t.Fatalf("Unexpected value")
	}
}

func Test_EndToEnd_LoadBalancer(t *testing.T) {
	host, ok := os.LookupEnv("RQLITE_GO_HTTP_E2E_HOST")
	if !ok {
		t.Skip("Skipping end-to-end test since no host is set")
	}
	ctx := context.Background()

	client, err := NewRoundRobinClient([]string{fmt.Sprintf("http://%s:4001", host)}, nil, 0, nil)
	if err != nil {
		t.Fatalf("Error creating round-robin client: %s", err)
	}
	defer client.Close()

	if _, err := client.ExecuteSingle(ctx, "DROP TABLE IF EXISTS lb_foo"); err != nil {
		t.Fatalf("Error dropping table: %s", err)
	}

	if _, err := client.ExecuteSingle(ctx, "CREATE TABLE lb_foo (id INT, name TEXT)"); err != nil {
		t.Fatalf("Error creating table: %s", err)
	}

	if _, err := client.ExecuteSingle(ctx, `INSERT INTO lb_foo(name) VALUES("fiona")`); err != nil {
		t.Fatalf("Error inserting row: %s", err)
	}

	stmt, err := NewSQLStatement("SELECT COUNT(*) FROM lb_foo")
	if err != nil {
		t.Fatalf("Error creating statement: %s", err)
	}

	resp, err := client.Query(ctx, SQLStatements{stmt}, nil)
	if err != nil {
		t.Fatalf("Error querying row count: %s", err)
	}

	results := resp.GetQueryResults()
	if len(results) != 1 {
		t.Fatalf("Unexpected number of results")
	}
	if len(results[0].Values) != 1 {
		t.Fatalf("Unexpected number of rows")
	}

	v, ok := results[0].Values[0][0].(json.Number)
	if !ok {
		t.Fatalf("Unexpected value type: %T", results[0].Values[0][0])
	}
	if v.String() != "1" {
		t.Fatalf("Unexpected value")
	}
}
