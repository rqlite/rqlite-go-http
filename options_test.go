package http

import (
	"reflect"
	"testing"
	"time"
)

// Test_MakeURLValues tests makeURLValues(). While not exported this
// functionality is key to this client library, so is unit-tested.
func Test_MakeURLValues(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		vals, err := makeURLValues(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vals) != 0 {
			t.Fatalf("expected empty values, got: %v", vals)
		}
	})

	t.Run("NilPointerValue", func(t *testing.T) {
		var foo *struct {
			X string `uvalue:"xxx"`
		}
		vals, err := makeURLValues(foo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vals) != 0 {
			t.Fatalf("expected empty values, got: %v", vals)
		}
	})

	t.Run("PointerToNonStruct", func(t *testing.T) {
		num := 42
		_, err := makeURLValues(&num)
		if err == nil {
			t.Error("expected an error when passing pointer to non-struct")
		}
	})

	t.Run("StructWithNoTags", func(t *testing.T) {
		type NoTags struct {
			A string
			B int
		}
		nt := &NoTags{"hello", 123}
		vals, err := makeURLValues(nt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vals) != 0 {
			t.Fatalf("expected no tagged fields, got: %v", vals)
		}
	})

	t.Run("StructWithSomeTags", func(t *testing.T) {
		type SomeTags struct {
			A string `uvalue:"aaa"`
			B int    // No tag
			C bool   `uvalue:"ccc"`
		}
		st := &SomeTags{
			A: "hello",
			B: 42,
			C: true,
		}
		vals, err := makeURLValues(st)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vals) != 2 {
			t.Fatalf("expected 2 values, got: %v", vals)
		}
		if got, want := vals.Get("aaa"), "hello"; got != want {
			t.Fatalf("expected aaa=%q, got %q", want, got)
		}
		// B is not tagged, should not appear
		if got := vals.Get("B"); got != "" {
			t.Fatalf("expected no value for \"B\", got %q", got)
		}
		if got, want := vals.Get("ccc"), "true"; got != want {
			t.Fatalf("expected ccc=%q, got %q", want, got)
		}
	})

	t.Run("StructWithAllSupportedTypes", func(t *testing.T) {
		type AllTypes struct {
			Str   string        `uvalue:"str"`
			Bln   bool          `uvalue:"bln"`
			IVal  int           `uvalue:"iVal"`
			I64   int64         `uvalue:"i64"`
			U64   uint64        `uvalue:"u64"`
			Dur   time.Duration `uvalue:"dur"`
			NoTag string        // This field is not tagged
		}
		at := &AllTypes{
			Str:   "sample",
			Bln:   true,
			IVal:  -999,
			I64:   1234567890123,
			U64:   9999999999999999999,
			Dur:   5 * time.Second,
			NoTag: "secret",
		}
		vals, err := makeURLValues(at)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(vals) != 6 {
			t.Fatalf("expected 6 values, got: %v", vals)
		}

		checks := []struct {
			key      string
			want     string
			wantDesc string
		}{
			{"str", "sample", "string field"},
			{"bln", "true", "bool field"},
			{"iVal", "-999", "int field"},
			{"i64", "1234567890123", "int64 field"},
			{"u64", "9999999999999999999", "uint64 field"},
			{"dur", "5s", "time.Duration field"},
		}
		for _, c := range checks {
			got := vals.Get(c.key)
			if got != c.want {
				t.Fatalf("expected %s to be %q, got %q", c.key, c.want, got)
			}
		}

		if vals.Get("NoTag") != "" {
			t.Fatalf("expected no value for \"NoTag\" (un-tagged field), got %q", vals.Get("NoTag"))
		}
	})

	t.Run("ZeroValues", func(t *testing.T) {
		type ZVals struct {
			S string `uvalue:"s"`
			I int    `uvalue:"i"`
			B bool   `uvalue:"b"`
		}
		z := &ZVals{}
		vals, err := makeURLValues(z)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := vals.Get("s"); got != "" {
			t.Fatalf("expected empty string for \"s\", got %q", got)
		}
		if got := vals.Get("i"); got != "0" {
			t.Fatalf("expected \"0\" for \"i\" (int zero), got %q", got)
		}
		if got := vals.Get("b"); got != "false" {
			t.Fatalf(`expected "false"  for "b", got %q`, got)
		}
	})

	t.Run("ZeroValues_OmitEmpy", func(t *testing.T) {
		type ZVals struct {
			S string `uvalue:"s,omitempty"`
			I int    `uvalue:"i,omitempty"`
			B bool   `uvalue:"b,omitempty"`
		}
		z := &ZVals{}
		vals, err := makeURLValues(z)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vals.Has("s") {
			t.Fatalf("expected no value for \"s\", got %q", vals.Get("s"))
		}
		if vals.Has("i") {
			t.Fatalf("expected no value for \"i\" (int zero), got %q", vals.Get("i"))
		}
		if vals.Has("b") {
			t.Fatalf(`expected no value for "b", got %q`, vals.Get("b"))
		}
	})

	t.Run("UnsupportedFieldType", func(t *testing.T) {
		type BadType struct {
			X float64 `uvalue:"x"`
		}
		b := &BadType{X: 3.14}
		vals, err := makeURLValues(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vals) != 0 {
			t.Fatalf("expected no values for float64, got: %v", vals)
		}
	})
}

// Test that the function signature matches our expectations
func Test_MakeURLValuesSignature(t *testing.T) {
	fn := reflect.ValueOf(MakeURLValues)
	if fn.Kind() != reflect.Func {
		t.Fatalf("makeURLValues is not a function")
	}
	if fn.Type().NumIn() != 1 {
		t.Fatalf("makeURLValues should have 1 input parameter, got %d", fn.Type().NumIn())
	}
	if fn.Type().NumOut() != 2 {
		t.Fatalf("makeURLValues should have 2 output parameters, got %d", fn.Type().NumOut())
	}
}
