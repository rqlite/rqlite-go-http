package http

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SQLStatement represents a single SQL statement, possibly with parameters.
type SQLStatement struct {
	// SQL is the text of the SQL statement, for example "INSERT INTO foo VALUES(?)".
	SQL string

	// PositionalParams is a slice of values for placeholders (?), if used.
	PositionalParams []any

	// NamedParams is a map of parameter names to values, if using named placeholders.
	NamedParams map[string]any
}

// NewSQLStatement creates a new SQLStatement from a SQL string and optional parameters.
// The parameters can be either a map of named parameters, or a slice of positional parameters.
func NewSQLStatement(stmt string, args ...any) (*SQLStatement, error) {
	s := SQLStatement{SQL: stmt}
	if len(args) == 0 {
		return &s, nil
	}

	if len(args) == 1 {
		if n, ok := args[0].(map[string]any); ok {
			s.NamedParams = n
			return &s, nil
		}
	}
	s.PositionalParams = args
	return &s, nil
}

// MarshalJSON implements a custom JSON representation so that SQL statements
// always appear as an array in the format rqlite expects.
func (s SQLStatement) MarshalJSON() ([]byte, error) {
	if len(s.NamedParams) > 0 {
		// e.g. ["INSERT INTO foo(name, age) VALUES(:name, :age)", { "name": "...", "age": ... }]
		arr := []any{s.SQL, s.NamedParams}
		return json.Marshal(arr)
	}

	if len(s.PositionalParams) > 0 {
		// e.g. ["INSERT INTO foo(name, age) VALUES(?, ?)", "param1", 123, ...]
		arr := make([]any, 1, 1+len(s.PositionalParams))
		arr[0] = s.SQL
		arr = append(arr, s.PositionalParams...)
		return json.Marshal(arr)
	}

	// No parameters => just return "SQL" as a JSON string.
	// e.g. "CREATE TABLE foo (id INTEGER NOT NULL ...)"
	return json.Marshal(s.SQL)
}

// UnmarshalJSON implements a custom JSON representation so that SQL statements
// always appear as an array in the format rqlite expects.
func (s *SQLStatement) UnmarshalJSON(data []byte) error {
	// create a JSON Decoder and tell is to UseNumber
	// so that it doesn't convert numbers to float64
	// which would be a lossy conversion
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var sql string
	if err := json.Unmarshal(data, &sql); err == nil {
		s.SQL = sql
		return nil
	}

	var arr []any
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) == 0 {
		return nil
	}

	sql, ok := arr[0].(string)
	if !ok {
		return fmt.Errorf("expected string for SQL statement, got %T", arr[0])
	}
	s.SQL = sql

	if len(arr) == 1 {
		return nil
	}

	// Remaining elements are either a single map, or positional parameters
	m, ok := arr[1].(map[string]any)
	if ok {
		s.NamedParams = m
	} else {
		s.PositionalParams = arr[1:]
	}
	return nil
}

// SQLStatements is a slice of SQLStatement.
type SQLStatements []*SQLStatement

func NewSQLStatementsFromStrings(stmts []string) SQLStatements {
	s := make(SQLStatements, len(stmts))
	for i, stmt := range stmts {
		s[i] = &SQLStatement{SQL: stmt}
	}
	return s
}

// MarshalJSON for SQLStatements produces a JSON array whose
// elements are each statement’s custom JSON form.
func (sts SQLStatements) MarshalJSON() ([]byte, error) {
	return json.Marshal([]*SQLStatement(sts))
}

func (sts *SQLStatements) UnmarshalJSON(data []byte) error {
	var stmts []*SQLStatement
	if err := json.Unmarshal(data, &stmts); err != nil {
		return err
	}
	s := make(SQLStatements, len(stmts))
	*sts = s
	for i, stmt := range stmts {
		s[i] = stmt
	}
	return nil
}
