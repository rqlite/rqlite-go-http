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

// Test_EndToEnd_StatementErrors issues statements against a non-existent table
// and checks that the resulting error is reported consistently under every
// error format rqlite supports, with and without PromoteErrors.
func Test_EndToEnd_StatementErrors(t *testing.T) {
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

	for _, stmt := range []string{
		"DROP TABLE IF EXISTS err_foo",
		"CREATE TABLE err_foo (id INTEGER PRIMARY KEY, name TEXT)",
	} {
		resp, err := client.ExecuteSingle(ctx, stmt)
		if err != nil {
			t.Fatalf("Error executing %q: %s", stmt, err)
		}
		if err := resp.Err(); err != nil {
			t.Fatalf("Error executing %q: %s", stmt, err)
		}
	}

	const missing = "no such table: nonexistent"

	// checkErr verifies that err reports the statement at index failing with
	// the missing table error. SQLite codes are expected unless the legacy
	// error format was requested, which carries only the message.
	checkErr := func(t *testing.T, err error, index int, sqlite bool) {
		t.Helper()
		want := fmt.Sprintf("statement %d: %s", index, missing)
		if err == nil || err.Error() != want {
			t.Fatalf("Expected error %q, got %v", want, err)
		}
		var stmtErr *StatementError
		if !errors.As(err, &stmtErr) {
			t.Fatalf("Expected error to wrap a StatementError, got %T", err)
		}
		if stmtErr.Message != missing {
			t.Fatalf("Expected message %q, got %q", missing, stmtErr.Message)
		}
		if !sqlite {
			if stmtErr.SQLite != nil {
				t.Fatalf("Expected no SQLite codes with legacy error format, got %+v", stmtErr.SQLite)
			}
			return
		}
		if stmtErr.SQLite == nil {
			t.Fatalf("Expected SQLite error codes, got none")
		}
		if stmtErr.SQLite.Code != 1 || stmtErr.SQLite.ExtendedCode != 1 {
			t.Fatalf("Unexpected SQLite error codes: %+v", stmtErr.SQLite)
		}
	}

	// checkPromoted verifies the error returned by the client itself, which is
	// nil unless errors are promoted.
	checkPromoted := func(t *testing.T, err error, promote bool, index int, sqlite bool) {
		t.Helper()
		if !promote {
			if err != nil {
				t.Fatalf("Expected nil error without promotion, got %v", err)
			}
			return
		}
		checkErr(t, err, index, sqlite)
	}

	formats := []struct {
		name   string
		format string
		sqlite bool
	}{
		{"default", "", true},
		{"v1", ErrorFormatV1, false},
		{"v2", ErrorFormatV2, true},
	}

	for _, f := range formats {
		for _, promote := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/promote=%v", f.name, promote), func(t *testing.T) {
				client.PromoteErrors(promote)

				// Execute: a good statement followed by a bad one.
				execResp, err := client.Execute(ctx, NewSQLStatementsFromStrings([]string{
					`INSERT INTO err_foo(name) VALUES("fiona")`,
					`INSERT INTO nonexistent(name) VALUES("fiona")`,
				}), &ExecuteOptions{Errors: f.format})
				checkPromoted(t, err, promote, 1, f.sqlite)
				checkErr(t, execResp.Err(), 1, f.sqlite)
				if len(execResp.Results) != 2 {
					t.Fatalf("Expected 2 execute results, got %d", len(execResp.Results))
				}
				if execResp.Results[0].Error != nil {
					t.Fatalf("Unexpected error on first execute result: %v", execResp.Results[0].Error)
				}
				if execResp.Results[1].Error == nil || execResp.Results[1].Error.Message != missing {
					t.Fatalf("Unexpected error on second execute result: %v", execResp.Results[1].Error)
				}

				// Query: a bad statement, in both result forms.
				for _, assoc := range []bool{false, true} {
					queryResp, err := client.Query(ctx, NewSQLStatementsFromStrings([]string{
						"SELECT * FROM nonexistent",
					}), &QueryOptions{Errors: f.format, Associative: assoc})
					checkPromoted(t, err, promote, 0, f.sqlite)
					checkErr(t, queryResp.Err(), 0, f.sqlite)
				}

				// Request: a good read followed by a bad write, in both result forms.
				for _, assoc := range []bool{false, true} {
					reqResp, err := client.Request(ctx, NewSQLStatementsFromStrings([]string{
						"SELECT COUNT(*) FROM err_foo",
						`INSERT INTO nonexistent(name) VALUES("fiona")`,
					}), &RequestOptions{Errors: f.format, Associative: assoc})
					checkPromoted(t, err, promote, 1, f.sqlite)
					checkErr(t, reqResp.Err(), 1, f.sqlite)
				}
			})
		}
	}
}
