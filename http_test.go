package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func Test_NewClient(t *testing.T) {
	client := NewClient("http://localhost:4001", nil)
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
	}))

	client := NewClient(ts.URL, nil)
	if err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth(username, password)
	authExp = true
	if err := client.Status(context.Background()); err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}

	client.SetBasicAuth("", "")
	authExp = false
	if err := client.Status(context.Background()); err != nil {
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
		// {
		// 	name:         "single CREATE TABLE statement",
		// 	statements:   NewSQLStatementsFromStrings([]string{"CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)"}),
		// 	opts:         &ExecuteOptions{Transaction: true, Timeout: mustParseDuration("1s")},
		// 	respBody:     `{"results": [{"last_insert_id": 123, "rows_affected": 456}]}`,
		// 	expURLValues: url.Values{"transaction": []string{"true"}, "timeout": []string{"1s"}},
		// },
		// {
		// 	name:       "two CREATE TABLE statements",
		// 	statements: NewSQLStatementsFromStrings([]string{"CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT)", "CREATE TABLE bar (id INTEGER PRIMARY KEY, name TEXT)"}),
		// 	opts:       nil,
		// 	respBody:   `{"results": [{"last_insert_id": 123, "rows_affected": 456}, {"last_insert_id": 789, "rows_affected": 101112}]}`,
		// },
		// {
		// 	name:         "single INSERT statement with positional arguments",
		// 	statements:   SQLStatements{SQLStatement{SQL: "INSERT INTO foo VALUES(?, ?)", PositionalParams: []any{"name", 123}}},
		// 	opts:         nil,
		// 	respBody:     `{"results": [{"last_insert_id": 123, "rows_affected": 456}]}`,
		// 	expURLValues: nil,
		// },
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
				fmt.Println(string(body))

				var gotStmts SQLStatements
				if err := json.Unmarshal(body, &gotStmts); err != nil {
					t.Fatalf("Unexpected error unmarshalling body: %v", err)
				}
				if !reflect.DeepEqual(tt.statements, gotStmts) {
					t.Fatalf("Expected %v, got %v", tt.statements, gotStmts)
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

			client := NewClient(ts.URL, nil)
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
