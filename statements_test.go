package http

import (
	"encoding/json"
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

func Test_SQLStatement_BLOBParameters(t *testing.T) {
	blob := []byte{0, 127, 255}
	for _, tc := range []struct {
		name string
		stmt SQLStatement
		want string
	}{
		{"positional", SQLStatement{SQL: "SELECT ?, ?, ?", PositionalParams: []any{blob, []byte{}, []byte(nil)}}, `["SELECT ?, ?, ?",[0,127,255],[],null]`},
		{"named", SQLStatement{SQL: "SELECT :a, :b, :c", NamedParams: map[string]any{"a": blob, "b": []byte{}, "c": []byte(nil)}}, `["SELECT :a, :b, :c",{"a":[0,127,255],"b":[],"c":null}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(&tc.stmt)
			if err != nil || string(b) != tc.want {
				t.Fatalf("JSON=%s error=%v want=%s", b, err, tc.want)
			}
			if !reflect.DeepEqual(blob, []byte{0, 127, 255}) {
				t.Fatal("caller BLOB was modified")
			}
			var original any
			if tc.stmt.NamedParams != nil {
				original = tc.stmt.NamedParams["a"]
			} else {
				original = tc.stmt.PositionalParams[0]
			}
			if _, ok := original.([]byte); !ok {
				t.Fatal("caller parameters were replaced")
			}
		})
	}
}
