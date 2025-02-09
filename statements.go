package http

import "encoding/json"

// SQLStatement represents a single SQL statement, possibly with parameters.
type SQLStatement struct {
	// SQL is the text of the SQL statement, for example "INSERT INTO foo VALUES(?)".
	SQL string

	// PositionalParams is a slice of values for placeholders (?), if used.
	PositionalParams []any

	// NamedParams is a map of parameter names to values, if using named placeholders.
	NamedParams map[string]any
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

// SQLStatements is a slice of SQLStatement.
type SQLStatements []SQLStatement

// MarshalJSON for SQLStatements produces a JSON array whose
// elements are each statement’s custom JSON form.
func (sts SQLStatements) MarshalJSON() ([]byte, error) {
	return json.Marshal([]SQLStatement(sts))
}
