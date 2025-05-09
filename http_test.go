package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func Test_NewClient(t *testing.T) {
	client, err := NewClient("http://localhost:4001", nil)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if client == nil {
		t.Error("Expected client to be non-nil")
	}
	if err := client.Close(); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func Test_BasicAuth(t *testing.T) {
	username := "user"
	password := "pass"

	authExp := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Fatalf("Unexpected path: %s", r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !authExp {
			if ok {
				t.Fatalf("basic auth should not be set")
			}
			return
		}

		if !ok {
			t.Fatalf("Expected BasicAuth to be set")
		}
		if exp, got := username, user; exp != got {
			t.Fatalf("Expected user to be '%s', got %s", exp, got)
		}
		if exp, got := password, pass; exp != got {
			t.Fatalf("Expected pass to be '%s', got %s", exp, got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))

	client, err := NewClient(ts.URL, nil)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	defer client.Close()
	if _, err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth(username, password)
	authExp = true
	if _, err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth("", "")
	authExp = false
	if _, err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}

func Test_Execute(t *testing.T) {
	for _, tt := range []struct {
		name         string
		statements   SQLStatements
		opts         *ExecuteOptions
		expURLValues url.Values
		respBody     string
	}{
		{
			name:       "single CREATE TABLE statement",
			statements: NewSQLStatementsFromStrings([]string{"CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)"}),
			opts:       nil,
			respBody:   `{"results": [{"last_insert_id": 123, "rows_affected": 456}]}`,
		},
		{
			name:         "single CREATE TABLE statement with options",
			statements:   NewSQLStatementsFromStrings([]string{"CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)"}),
			opts:         &ExecuteOptions{Transaction: true, Timeout: mustParseDuration("1s")},
			respBody:     `{"results": [{"last_insert_id": 123, "rows_affected": 456}]}`,
			expURLValues: url.Values{"transaction": []string{"true"}, "timeout": []string{"1s"}},
		},
		{
			name:       "two CREATE TABLE statements",
			statements: NewSQLStatementsFromStrings([]string{"CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)", "CREATE TABLE bar (id INTEGER PRIMARY KEY, name TEXT)"}),
			opts:       nil,
			respBody:   `{"results": [{"last_insert_id": 123, "rows_affected": 456}, {"last_insert_id": 789, "rows_affected": 101112}]}`,
		},
		{
			name:         "single INSERT statement with positional arguments",
			statements:   SQLStatements{&SQLStatement{SQL: "INSERT INTO foo VALUES(?, ?)", PositionalParams: []any{"name", float64(123)}}},
			opts:         nil,
			respBody:     `{"results": [{"last_insert_id": 123, "rows_affected": 456}]}`,
			expURLValues: nil,
		},
		{
			name:         "single INSERT statement with named arguments",
			statements:   SQLStatements{&SQLStatement{SQL: "INSERT INTO foo VALUES(:name, :age)", NamedParams: map[string]any{"name": "name", "age": float64(123)}}},
			opts:         nil,
			respBody:     `{"results": [{"last_insert_id": 123, "rows_affected": 456}]}`,
			expURLValues: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/db/execute" {
					t.Fatalf("Unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Fatalf("Expected POST, got %s", r.Method)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Unexpected error reading body: %v", err)
				}
				defer r.Body.Close()

				var gotStmts SQLStatements
				if err := json.Unmarshal(body, &gotStmts); err != nil {
					t.Fatalf("Unexpected error unmarshalling body: %v", err)
				}
				if !reflect.DeepEqual(tt.statements, gotStmts) {
					t.Fatalf("Expected '%v' in request body, got '%v'", tt.statements, gotStmts)
				}

				if tt.expURLValues != nil {
					values, err := url.ParseQuery(r.URL.RawQuery)
					if err != nil {
						t.Fatalf("Unexpected error parsing query string: %s", r.URL.RawQuery)
					}
					if !reflect.DeepEqual(tt.expURLValues, values) {
						t.Fatalf("Expected %v, got %v", tt.expURLValues, r.URL.Query())
					}
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.respBody))
			}))
			defer ts.Close()

			client, err := NewClient(ts.URL, nil)
			if err != nil {
				t.Fatalf("Expected nil error, got %v", err)
			}
			defer client.Close()
			gotER, err := client.Execute(context.Background(), tt.statements, tt.opts)
			if err != nil {
				t.Fatalf("Expected nil error, got %v", err)
			}
			expER := mustUnmarshalExecuteResponse(tt.respBody)

			if reflect.DeepEqual(expER, gotER) {
				t.Fatalf("Expected %+v, got %+v", expER, gotER)
			}
		})
	}
}

func Test_Query(t *testing.T) {
	tests := []struct {
		name         string
		statements   SQLStatements
		opts         QueryOptions
		expURLValues url.Values
		respBody     string
	}{
		{
			name:         "simple SELECT query",
			statements:   NewSQLStatementsFromStrings([]string{"SELECT * FROM foo"}),
			opts:         QueryOptions{},
			expURLValues: nil,
			respBody:     `{"results": [{"columns": ["id", "name"], "values": [[1, "Alice"], [2, "Bob"]]}], "time": 0.456}`,
		},
		{
			name:       "SELECT query with options",
			statements: NewSQLStatementsFromStrings([]string{"SELECT name FROM bar"}),
			opts: QueryOptions{
				Pretty:  true,
				Timeout: mustParseDuration("2s"),
			},
			expURLValues: url.Values{
				"pretty":  []string{"true"},
				"timeout": []string{"2s"},
			},
			respBody: `{"results": [{"columns": ["name"], "values": [["Charlie"]]}], "time": 0.789}`,
		},
		{
			name:         "multiple SELECT queries",
			statements:   NewSQLStatementsFromStrings([]string{"SELECT 1", "SELECT 2"}),
			opts:         QueryOptions{},
			expURLValues: nil,
			respBody:     `{"results": [{"columns": ["?column?"], "values": [[1]]}, {"columns": ["?column?"], "values": [[2]]}], "time": 1.234}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/db/query" {
					t.Fatalf("Unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Fatalf("Expected POST, got %s", r.Method)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Unexpected error reading body: %v", err)
				}
				defer r.Body.Close()

				var gotStmts SQLStatements
				if err := json.Unmarshal(body, &gotStmts); err != nil {
					t.Fatalf("Unexpected error unmarshalling body: %v", err)
				}
				if !reflect.DeepEqual(tt.statements, gotStmts) {
					t.Fatalf("Expected statements %+v, got %+v", tt.statements, gotStmts)
				}

				if tt.expURLValues != nil {
					values, err := url.ParseQuery(r.URL.RawQuery)
					if err != nil {
						t.Fatalf("Unexpected error parsing query string: %s", r.URL.RawQuery)
					}
					if !reflect.DeepEqual(tt.expURLValues, values) {
						t.Fatalf("Expected URL values %v, got %v", tt.expURLValues, values)
					}
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.respBody))
			}))
			defer ts.Close()

			client, err := NewClient(ts.URL, nil)
			if err != nil {
				t.Fatalf("Expected nil error, got %v", err)
			}
			defer client.Close()
			gotQR, err := client.Query(context.Background(), tt.statements, &tt.opts)
			if err != nil {
				t.Fatalf("Expected nil error, got %v", err)
			}
			expQR := mustUnmarshalQueryResponse(tt.respBody)

			if !reflect.DeepEqual(expQR, *gotQR) {
				t.Fatalf("Expected %+v, got %+v", expQR, gotQR)
			}
		})
	}
}

func Test_QueryAssoc(t *testing.T) {
	tests := []struct {
		name         string
		statements   SQLStatements
		opts         QueryOptions
		expURLValues url.Values
		respBody     string
	}{
		{
			name:         "simple SELECT query",
			statements:   NewSQLStatementsFromStrings([]string{"SELECT * FROM foo"}),
			opts:         QueryOptions{},
			expURLValues: nil,
			respBody:     `{"results":[{"types":{"id":"integer","name":"text"},"rows":[{"id":1,"name":"fiona"}],"time":0.1}],"time":0.2}`,
		},
		{
			name:       "SELECT query with options",
			statements: NewSQLStatementsFromStrings([]string{"SELECT name FROM bar"}),
			opts: QueryOptions{
				Pretty:  true,
				Timeout: mustParseDuration("2s"),
			},
			expURLValues: url.Values{
				"pretty":      []string{"true"},
				"timeout":     []string{"2s"},
				"associative": []string{"true"},
			},
			respBody: `{"results":[{"types":{"id":"integer","name":"text"},"rows":[{"id":1,"name":"fiona"}],"time":0.1}],"time":0.2}`,
		},
		{
			name:         "multiple SELECT queries",
			statements:   NewSQLStatementsFromStrings([]string{"SELECT 1", "SELECT 2"}),
			opts:         QueryOptions{},
			expURLValues: nil,
			respBody:     `{"results":[{"types":{"id":"integer","name":"text"},"rows":[{"id":1,"name":"fiona"}],"time":0.1}],"time":0.2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/db/query" {
					t.Fatalf("Unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Fatalf("Expected POST, got %s", r.Method)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Unexpected error reading body: %v", err)
				}
				defer r.Body.Close()

				var gotStmts SQLStatements
				if err := json.Unmarshal(body, &gotStmts); err != nil {
					t.Fatalf("Unexpected error unmarshalling body: %v", err)
				}
				if !reflect.DeepEqual(tt.statements, gotStmts) {
					t.Fatalf("Expected statements %+v, got %+v", tt.statements, gotStmts)
				}

				if tt.expURLValues != nil {
					values, err := url.ParseQuery(r.URL.RawQuery)
					if err != nil {
						t.Fatalf("Unexpected error parsing query string: %s", r.URL.RawQuery)
					}
					if !reflect.DeepEqual(tt.expURLValues, values) {
						t.Fatalf("Expected URL values %v, got %v", tt.expURLValues, values)
					}
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.respBody))
			}))
			defer ts.Close()

			client, err := NewClient(ts.URL, nil)
			if err != nil {
				t.Fatalf("Expected nil error, got %v", err)
			}
			defer client.Close()
			tt.opts.Associative = true
			gotQR, err := client.Query(context.Background(), tt.statements, &tt.opts)
			if err != nil {
				t.Fatalf("Expected nil error, got %v", err)
			}
			expQR := mustUnmarshalQueryResponse(tt.respBody)

			if !reflect.DeepEqual(expQR, *gotQR) {
				t.Fatalf("Expected %+v, got %+v", expQR, gotQR)
			}
		})
	}
}

func Test_Request(t *testing.T) {
	statements := SQLStatements{
		{
			SQL:              "INSERT INTO foo(name) VALUES(?)",
			PositionalParams: []interface{}{"alice"},
		},
		{
			SQL:         "SELECT * FROM foo WHERE name=:name",
			NamedParams: map[string]interface{}{"name": "bob"},
		},
	}

	opts := RequestOptions{
		Transaction: true,
		Pretty:      true,
	}

	responseJSON := `{
		"results": [
			{
				"last_insert_id": 1,
				"rows_affected": 1,
				"time": 0.001
			},
			{
				"columns": ["id","name"],
				"types": ["integer","text"],
				"values": [[1,"alice"]],
				"time": 0.002
			}
		],
		"time": 0.003
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/db/request" {
			t.Errorf("expected path /db/request, got %s", r.URL.Path)
		}

		q := r.URL.Query()
		if _, ok := q["transaction"]; !ok {
			t.Error("expected ?transaction=... to be present, but not found")
		}
		if _, ok := q["pretty"]; !ok {
			t.Error("expected ?pretty=... to be present, but not found")
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %v", err)
		}

		var gotStmts SQLStatements
		if err := json.Unmarshal(bodyBytes, &gotStmts); err != nil {
			t.Fatalf("failed to unmarshal posted JSON: %v", err)
		}

		if !reflect.DeepEqual(statements, gotStmts) {
			t.Errorf("unexpected statements: expected: %v, got %v", statements, gotStmts)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	resp, err := cl.Request(context.Background(), statements, &opts)
	if err != nil {
		t.Fatalf("unexpected error from Request: %v", err)
	}

	results, ok := resp.Results.([]RequestResult)
	if !ok {
		t.Fatalf("unexpected type for Results: %T", resp.Results)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if resp.Time != 0.003 {
		t.Errorf("expected Time=0.003, got %f", resp.Time)
	}

	first := results[0]
	if first.LastInsertID == nil || *first.LastInsertID != 1 {
		t.Errorf("expected last_insert_id=1, got %v", first.LastInsertID)
	}
	if first.RowsAffected == nil || *first.RowsAffected != 1 {
		t.Errorf("expected rows_affected=1, got %v", first.RowsAffected)
	}
	if first.Time != 0.001 {
		t.Errorf("expected time=0.001, got %f", first.Time)
	}

	second := results[1]
	if len(second.Columns) != 2 || second.Columns[0] != "id" || second.Columns[1] != "name" {
		t.Errorf("expected columns=[\"id\",\"name\"], got %v", second.Columns)
	}
	if len(second.Values) != 1 || len(second.Values[0]) != 2 {
		t.Errorf("unexpected values: %v", second.Values)
	}
	if second.Time != 0.002 {
		t.Errorf("expected time=0.002, got %f", second.Time)
	}
}

func Test_RequestAssoc(t *testing.T) {
	statements := SQLStatements{
		{
			SQL:              "INSERT INTO foo(name) VALUES(?)",
			PositionalParams: []interface{}{"alice"},
		},
		{
			SQL:         "SELECT * FROM foo WHERE name=:name",
			NamedParams: map[string]interface{}{"name": "bob"},
		},
	}

	opts := RequestOptions{
		Transaction: true,
		Pretty:      true,
	}

	responseJSON := `{
		"results": [
			{
				"last_insert_id": 1,
				"rows_affected": 1,
				"time": 0.001
			},
			{
				"types": {"id": "integer", "name": "text"},
				"rows": [
					{ "id": 1, "name": "alice"}
				],
				"time": 0.002
			}
		],
		"time": 0.003
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/db/request" {
			t.Errorf("expected path /db/request, got %s", r.URL.Path)
		}

		q := r.URL.Query()
		if _, ok := q["transaction"]; !ok {
			t.Error("expected ?transaction=... to be present, but not found")
		}
		if _, ok := q["pretty"]; !ok {
			t.Error("expected ?pretty=... to be present, but not found")
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %v", err)
		}

		var gotStmts SQLStatements
		if err := json.Unmarshal(bodyBytes, &gotStmts); err != nil {
			t.Fatalf("failed to unmarshal posted JSON: %v", err)
		}

		if !reflect.DeepEqual(statements, gotStmts) {
			t.Errorf("unexpected statements: expected: %v, got %v", statements, gotStmts)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	resp, err := cl.Request(context.Background(), statements, &opts)
	if err != nil {
		t.Fatalf("unexpected error from Request: %v", err)
	}

	results, ok := resp.Results.([]RequestResultAssoc)
	if !ok {
		t.Fatalf("unexpected type for Results: %T", resp.Results)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if resp.Time != 0.003 {
		t.Errorf("expected Time=0.003, got %f", resp.Time)
	}

	first := results[0]
	if first.LastInsertID == nil || *first.LastInsertID != 1 {
		t.Errorf("expected last_insert_id=1, got %v", first.LastInsertID)
	}
	if first.RowsAffected == nil || *first.RowsAffected != 1 {
		t.Errorf("expected rows_affected=1, got %v", first.RowsAffected)
	}
	if first.Time != 0.001 {
		t.Errorf("expected time=0.001, got %f", first.Time)
	}

	second := results[1]
	if !reflect.DeepEqual(second.Types, map[string]string{"id": "integer", "name": "text"}) {
		t.Errorf("expected types={\"id\":\"integer\",\"name\":\"text\"}, got %v", second.Types)
	}
	if !reflect.DeepEqual(second.Rows, []map[string]any{{"id": float64(1), "name": "alice"}}) {
		t.Errorf("unexpected rows: %v", second.Rows)
	}
	if second.Time != 0.002 {
		t.Errorf("expected time=0.002, got %f", second.Time)
	}
}

func Test_PromoteErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": [{"error": "some error"}]}`))
	}))
	defer ts.Close()

	client, err := NewClient(ts.URL, nil)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	defer client.Close()

	_, err = client.Execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	_, err = client.Query(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	_, err = client.Request(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.PromoteErrors(true)

	_, err = client.Execute(context.Background(), nil, nil)
	if err == nil {
		t.Fatalf("Expected non-nil error after promoting errors, got nil")
	}
	_, err = client.Query(context.Background(), nil, nil)
	if err == nil {
		t.Fatalf("Expected non-nil error after promoting errors, got nil")
	}
	_, err = client.Request(context.Background(), nil, nil)
	if err == nil {
		t.Fatalf("Expected non-nil error after promoting errors, got nil")
	}
}

func Test_Boot(t *testing.T) {
	expectedData := []byte("some raw SQLite bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/boot" {
			t.Errorf("expected path /boot, got %s", r.URL.Path)
		}

		postedData, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %v", err)
		}

		if !bytes.Equal(postedData, expectedData) {
			t.Errorf("posted data does not match.\nwant: %q\ngot:  %q", expectedData, postedData)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	dataReader := bytes.NewReader(expectedData)
	err = cl.Boot(context.Background(), dataReader)
	if err != nil {
		t.Fatalf("unexpected error calling Boot: %v", err)
	}
}

func Test_Backup(t *testing.T) {
	expectedData := []byte("some random bytes")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/db/backup" {
			t.Errorf("expected path /db/backup, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(expectedData); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	rc, err := cl.Backup(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error calling Backup: %v", err)
	}
	defer rc.Close()

	actualData, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("unexpected error reading backup data: %v", err)
	}

	if string(actualData) != string(expectedData) {
		t.Errorf("mismatched backup data.\nwant: %q\ngot:  %q", expectedData, actualData)
	}
}

func Test_Status(t *testing.T) {
	expectedData := []byte(`{"foo":"bar"}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Errorf("expected path /status, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(expectedData); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	rawMsg, err := cl.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error calling Status: %v", err)
	}

	if string(rawMsg) != string(expectedData) {
		t.Errorf("mismatched Status data.\nwant: %q\ngot:  %q", expectedData, rawMsg)
	}
}

func Test_Expvar(t *testing.T) {
	expectedData := []byte(`{"expvar_key":"expvar_value"}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/debug/vars" {
			t.Errorf("expected path /expvar, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(expectedData); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	rawMsg, err := cl.Expvar(context.Background())
	if err != nil {
		t.Fatalf("unexpected error calling Expvar: %v", err)
	}

	if string(rawMsg) != string(expectedData) {
		t.Errorf("mismatched Expvar data.\nwant: %q\ngot:  %q", expectedData, rawMsg)
	}
}

func Test_Nodes(t *testing.T) {
	expectedData := []byte(`[{"api_addr":"localhost:4001","reachable":true}]`)
	expectedRawQuery := "nonvoters=true"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nodes" {
			t.Errorf("expected path /nodes, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != expectedRawQuery {
			t.Errorf("expected query %s, got %s", expectedRawQuery, r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(expectedData); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	rawMsg, err := cl.Nodes(context.Background(), &NodeOptions{NonVoters: true})
	if err != nil {
		t.Fatalf("unexpected error calling Nodes: %v", err)
	}

	if string(rawMsg) != string(expectedData) {
		t.Errorf("mismatched Nodes data.\nwant: %q\ngot:  %q", expectedData, rawMsg)
	}
}

func Test_RemoveNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/remove" {
			t.Errorf("expected path /remove, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %v", err)
		}
		if string(b) != `{"id":"id1"}` {
			t.Errorf("unexpected request body: %q", b)
		}
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	if err := cl.RemoveNode(context.Background(), "id1"); err != nil {
		t.Fatalf("unexpected error calling RemoveNode: %v", err)
	}
}

func Test_Version(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-RQLITE-VERSION", "1.2.3")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	v, err := cl.Version(context.Background())
	if err != nil {
		t.Fatalf("unexpected error retrieving status: %v", err)
	}

	if exp, got := "1.2.3", v; exp != got {
		t.Fatalf("wrong version, exp: %s, got: %s", exp, got)
	}
}

func Test_Ready(t *testing.T) {
	expectedData := []byte(`[+]node ok`)
	expectedRawQuery := "sync=true"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Errorf("expected path /readyz, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != expectedRawQuery {
			t.Errorf("expected query %s, got %s", expectedRawQuery, r.URL.RawQuery)
		}
		if _, err := w.Write(expectedData); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	cl, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error from NewClient: %v", err)
	}
	v, err := cl.Ready(context.Background(), &ReadyOptions{Sync: true})
	if err != nil {
		t.Fatalf("unexpected error retrieving status: %v", err)
	}
	if string(v) != string(expectedData) {
		t.Fatalf("mismatched Ready response.\nwant: %q\ngot:  %q", expectedData, v)
	}
}

func mustUnmarshalQueryResponse(s string) QueryResponse {
	var qr QueryResponse
	if err := json.Unmarshal([]byte(s), &qr); err != nil {
		panic(err)
	}
	return qr
}

func mustUnmarshalExecuteResponse(s string) ExecuteResponse {
	var er ExecuteResponse
	if err := json.Unmarshal([]byte(s), &er); err != nil {
		panic(err)
	}
	return er
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
