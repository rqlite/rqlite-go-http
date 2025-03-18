package http

import (
	"reflect"
	"testing"
)

func Test_NewSQLStatementFrom_Positional(t *testing.T) {
	for i, tt := range []struct {
		stmt string
		args []any
		want *SQLStatement
	}{
		{
			stmt: "SELECT * FROM foo",
			want: &SQLStatement{SQL: "SELECT * FROM foo"},
		},
		{
			stmt: "SELECT * FROM foo WHERE id = ?",
			args: []any{42},
			want: &SQLStatement{SQL: "SELECT * FROM foo WHERE id = ?", PositionalParams: []any{42}},
		},
		{
			stmt: "SELECT * FROM foo WHERE id = ? AND name = ?",
			args: []any{42, "hello"},
			want: &SQLStatement{SQL: "SELECT * FROM foo WHERE id = ? AND name = ?", PositionalParams: []any{42, "hello"}},
		},
	} {
		got, err := NewSQLStatement(tt.stmt, tt.args...)
		if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("[%d] got: %v, want: %v", i, got, tt.want)
		}
	}
}

func Test_NewSQLStatementFrom_Named(t *testing.T) {
	got, err := NewSQLStatement("SELECT * FROM foo WHERE id = :id AND name = :name", map[string]any{
		"id":   42,
		"name": "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := &SQLStatement{SQL: "SELECT * FROM foo WHERE id = :id AND name = :name", NamedParams: map[string]any{
		"id":   42,
		"name": "hello",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: %v, want: %v", got, want)
	}
}
