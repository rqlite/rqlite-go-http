package stdlib_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
	"github.com/rqlite/rqlite-go-http/stdlib"
	_ "github.com/rqlite/rqlite-go-http/stdlib/rqlite"
)

func init() {
	sql.Register("rqlite-lb", &stdlib.Driver{LoadBalancer: failingLoadBalancer{}})
	sql.Register("rqlite-allowquery", &stdlib.Driver{AllowQueryInTxn: true})
}

var _ rqlitehttp.LoadBalancer = (*failingLoadBalancer)(nil)

type failingLoadBalancer struct{}

func (_ failingLoadBalancer) Next() (*url.URL, error) {
	return nil, rqlitehttp.ErrNoHostsAvailable
}

func testDatabase(t *testing.T, driver ...string) *sql.DB {
	t.Helper()
	host, ok := os.LookupEnv("RQLITE_GO_HTTP_E2E_HOST")
	if !ok {
		t.Skip("Skipping: RQLITE_GO_HTTP_E2E_HOST not set")
	}
	driverName := "rqlite"
	if len(driver) > 0 {
		driverName = driver[0]
	}
	db, err := sql.Open(driverName, fmt.Sprintf("http://%s:4001", host))
	if err != nil {
		t.Fatalf("sql.Open: %s", err)
	}
	return db
}

func Test_EndToEnd_SQLDriver(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_sqldriver")
	defer db.Exec("DROP TABLE IF EXISTS test_sqldriver")

	if _, err := db.Exec("CREATE TABLE test_sqldriver (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("INSERT INTO test_sqldriver (id, name) VALUES (1, 'foo')"); err != nil {
		t.Fatal(err)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM test_sqldriver WHERE id = 1").Scan(&name); err != nil {
		t.Fatal(err)
	} else if name != "foo" {
		t.Fatalf("expected %q, got %q", "foo", name)
	}
}

func Test_EndToEnd_SQLDriverWithLoadBalancer(t *testing.T) {
	db := testDatabase(t, "rqlite-lb")
	defer db.Close()

	if _, err := db.Exec("SELECT 1"); !errors.Is(err, rqlitehttp.ErrNoHostsAvailable) {
		t.Fatal("expected ErrNoHostsAvailable")
	}
}

func Test_EndToEnd_SQLDriver_MultipleRows(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_sqldriver_multiplerows")
	defer db.Exec("DROP TABLE IF EXISTS test_sqldriver_multiplerows")

	if _, err := db.Exec("CREATE TABLE test_sqldriver_multiplerows (id INTEGER PRIMARY KEY, username TEXT, age INTEGER)"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("INSERT INTO test_sqldriver_multiplerows (id, username, age) VALUES (1, 'alice', 30)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO test_sqldriver_multiplerows (id, username, age) VALUES (2, 'bob', 25)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO test_sqldriver_multiplerows (id, username, age) VALUES (3, 'charlie', 35)"); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("SELECT id, username, age FROM test_sqldriver_multiplerows ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var users []struct {
		ID       int
		Username string
		Age      int
	}

	for rows.Next() {
		var u struct {
			ID       int
			Username string
			Age      int
		}
		if err := rows.Scan(&u.ID, &u.Username, &u.Age); err != nil {
			t.Fatal(err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	} else if users[0].Username != "alice" {
		t.Fatalf("expected %q, got %q", "alice", users[0].Username)
	} else if users[1].Username != "bob" {
		t.Fatalf("expected %q, got %q", "bob", users[1].Username)
	} else if users[2].Username != "charlie" {
		t.Fatalf("expected %q, got %q", "charlie", users[2].Username)
	}
}

func Test_EndToEnd_SQLDataTypes(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_sqldatatypes")
	defer db.Exec("DROP TABLE IF EXISTS test_sqldatatypes")

	if _, err := db.Exec(`CREATE TABLE test_sqldatatypes (
		id INTEGER PRIMARY KEY,
		int_val INTEGER,
		float_val REAL,
		bool_val INTEGER,
		string_val TEXT,
		blob_val BLOB,
		time_val TEXT,
		nullable_int INTEGER,
		nullable_string TEXT,
		nullable_float REAL
	) STRICT`); err != nil {
		t.Fatal(err)
	}

	var idCounter int64

	t.Run("Integer Types", func(t *testing.T) {
		tests := []struct {
			name     string
			value    int64
			scanType string // which type to scan into
		}{
			// Basic values
			{"zero int64", 0, "int64"},
			{"zero int32", 0, "int32"},
			{"zero int16", 0, "int16"},
			{"zero int8", 0, "int8"},
			{"zero int", 0, "int"},
			{"positive int64", 42, "int64"},
			{"positive int32", 42, "int32"},
			{"positive int", 42, "int"},
			{"negative int64", -42, "int64"},
			{"negative int32", -42, "int32"},
			{"negative int", -42, "int"},

			// Boundary values for int8
			{"max int8", 127, "int8"},
			{"min int8", -128, "int8"},

			// Boundary values for int16
			{"max int16", 32767, "int16"},
			{"min int16", -32768, "int16"},

			// Boundary values for int32
			{"max int32", 2147483647, "int32"},
			{"min int32", -2147483648, "int32"},

			// Boundary values for int64
			{"max int64", 9223372036854775807, "int64"},
			{"min int64", -9223372036854775808, "int64"},

			// Unsigned integers
			{"max uint8", 255, "uint8"},
			{"max uint16", 65535, "uint16"},
			{"max uint32", 4294967295, "uint32"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				idCounter++
				currentID := idCounter

				if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, int_val) VALUES (?, ?)", currentID, tt.value); err != nil {
					t.Fatal(err)
				}

				var scanned any
				switch tt.scanType {
				case "int64":
					var v int64
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != tt.value {
						t.Fatalf("expected %v, got %v", tt.value, v)
					}
					scanned = v
				case "int32":
					var v int32
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != int32(tt.value) {
						t.Fatalf("expected %v, got %v", int32(tt.value), v)
					}
					scanned = v
				case "int16":
					var v int16
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != int16(tt.value) {
						t.Fatalf("expected %v, got %v", int16(tt.value), v)
					}
					scanned = v
				case "int8":
					var v int8
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != int8(tt.value) {
						t.Fatalf("expected %v, got %v", int8(tt.value), v)
					}
					scanned = v
				case "int":
					var v int
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != int(tt.value) {
						t.Fatalf("expected %v, got %v", int(tt.value), v)
					}
					scanned = v
				case "uint8":
					var v uint8
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != uint8(tt.value) {
						t.Fatalf("expected %v, got %v", uint8(tt.value), v)
					}
					scanned = v
				case "uint16":
					var v uint16
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != uint16(tt.value) {
						t.Fatalf("expected %v, got %v", uint16(tt.value), v)
					}
					scanned = v
				case "uint32":
					var v uint32
					if err := db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != uint32(tt.value) {
						t.Fatalf("expected %v, got %v", uint32(tt.value), v)
					}
					scanned = v
				}
				_ = scanned
			})
		}
	})

	t.Run("Floating Point Types", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float64
			scanType  string
			checkBits bool
		}{
			{"zero float64", 0.0, "float64", true},
			{"zero float32", 0.0, "float32", false},
			{"simple decimal", 123.456, "float64", true},
			{"very small", 0.000001, "float64", true},
			{"negative", -123.456, "float64", true},
			{"pi precision", 3.141592653589793, "float64", true},
			{"integer as float", 1.0, "float64", true},
			{"large value", 1.797693e+308, "float64", true},
			{"float32 value", 3.402823e+38, "float32", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				idCounter++
				currentID := idCounter

				if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, float_val) VALUES (?, ?)", currentID, tt.value); err != nil {
					t.Fatal(err)
				}

				switch tt.scanType {
				case "float64":
					var v float64
					if err := db.QueryRow("SELECT float_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if tt.checkBits {
						if math.Float64bits(tt.value) != math.Float64bits(v) {
							t.Fatalf("precision lost for %s: want bits %016x, got %016x", tt.name, math.Float64bits(tt.value), math.Float64bits(v))
						}
					} else if v != tt.value {
						t.Fatalf("expected %v, got %v", tt.value, v)
					}
				case "float32":
					var v float32
					if err := db.QueryRow("SELECT float_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
						t.Fatal(err)
					} else if v != float32(tt.value) {
						t.Fatalf("expected %v, got %v", float32(tt.value), v)
					}
				}
			})
		}
	})

	t.Run("Boolean Types", func(t *testing.T) {
		tests := []struct {
			name     string
			value    int64
			expected bool
		}{
			{"true", 1, true},
			{"false", 0, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				idCounter++
				currentID := idCounter

				if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, bool_val) VALUES (?, ?)", currentID, tt.value); err != nil {
					t.Fatal(err)
				}

				var v bool
				if err := db.QueryRow("SELECT bool_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
					t.Fatal(err)
				} else if v != tt.expected {
					t.Fatalf("expected %v, got %v", tt.expected, v)
				}
			})
		}
	})

	t.Run("String Types", func(t *testing.T) {
		longString := strings.Repeat("abcdefghij", 1024) // 10KB

		tests := []struct {
			name  string
			value string
		}{
			{"empty string", ""},
			{"simple ascii", "hello world"},
			{"unicode", "Hello 世界 🎉"},
			{"special chars", "line1\nline2\ttab"},
			{"long string", longString},
			{"sql injection chars", "'; DROP TABLE--"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				idCounter++
				currentID := idCounter

				if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, string_val) VALUES (?, ?)", currentID, tt.value); err != nil {
					t.Fatal(err)
				}

				var v string
				if err := db.QueryRow("SELECT string_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v); err != nil {
					t.Fatal(err)
				} else if v != tt.value {
					t.Fatalf("expected %q, got %q", tt.value, v)
				}
			})
		}
	})

	t.Run("Binary Types", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS test_sqldatatypes_blob")
		defer db.Exec("DROP TABLE IF EXISTS test_sqldatatypes_blob")

		if _, err := db.Exec(`CREATE TABLE test_sqldatatypes_blob (
			id INTEGER PRIMARY KEY,
			blob_val BLOB
		)`); err != nil {
			t.Fatal(err)
		}

		allBytes := make([]byte, 256)
		for i := range 256 {
			allBytes[i] = byte(i)
		}

		largeBinary := make([]byte, 1024*1024)
		for i := range largeBinary {
			largeBinary[i] = byte(i % 256)
		}

		tests := []struct {
			name  string
			value []byte
		}{
			{"empty blob", []byte{}},
			{"small binary", []byte{0x00, 0x01, 0xFF, 0xFE}},
			{"all bytes", allBytes},
			{"large binary", largeBinary},
		}

		blobID := int64(1)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				currentID := blobID
				blobID++

				if _, err := db.Exec("INSERT INTO test_sqldatatypes_blob (id, blob_val) VALUES (?, ?)", currentID, tt.value); err != nil {
					t.Fatal(err)
				}

				var v []byte
				if err := db.QueryRow("SELECT blob_val FROM test_sqldatatypes_blob WHERE id = ?", currentID).Scan(&v); err != nil {
					t.Fatal(err)
				} else if !bytes.Equal(tt.value, v) {
					t.Fatalf("blob mismatch: len(want)=%d, len(got)=%d", len(tt.value), len(v))
				}
			})
		}
	})

	t.Run("NULL Handling", func(t *testing.T) {
		t.Run("sql.Null* types", func(t *testing.T) {
			idCounter++
			nullID := idCounter

			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, nullable_int, nullable_string, nullable_float) VALUES (?, NULL, NULL, NULL)", nullID); err != nil {
				t.Fatal(err)
			}

			var nullInt sql.NullInt64
			if err := db.QueryRow("SELECT nullable_int FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullInt); err != nil {
				t.Fatal(err)
			} else if nullInt.Valid {
				t.Fatal("expected NULL for int")
			}

			var nullStr sql.NullString
			if err := db.QueryRow("SELECT nullable_string FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullStr); err != nil {
				t.Fatal(err)
			} else if nullStr.Valid {
				t.Fatal("expected NULL for string")
			}

			var nullFloat sql.NullFloat64
			if err := db.QueryRow("SELECT nullable_float FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullFloat); err != nil {
				t.Fatal(err)
			} else if nullFloat.Valid {
				t.Fatal("expected NULL for float")
			}

			idCounter++
			nonNullID := idCounter
			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, nullable_int, nullable_string, nullable_float) VALUES (?, ?, ?, ?)",
				nonNullID, 42, "test", 3.14); err != nil {
				t.Fatal(err)
			}

			if err := db.QueryRow("SELECT nullable_int FROM test_sqldatatypes WHERE id = ?", nonNullID).Scan(&nullInt); err != nil {
				t.Fatal(err)
			} else if !nullInt.Valid {
				t.Fatal("expected valid int")
			} else if nullInt.Int64 != 42 {
				t.Fatalf("expected 42, got %d", nullInt.Int64)
			}

			if err := db.QueryRow("SELECT nullable_string FROM test_sqldatatypes WHERE id = ?", nonNullID).Scan(&nullStr); err != nil {
				t.Fatal(err)
			} else if !nullStr.Valid {
				t.Fatal("expected valid string")
			} else if nullStr.String != "test" {
				t.Fatalf("expected %q, got %q", "test", nullStr.String)
			}

			if err := db.QueryRow("SELECT nullable_float FROM test_sqldatatypes WHERE id = ?", nonNullID).Scan(&nullFloat); err != nil {
				t.Fatal(err)
			} else if !nullFloat.Valid {
				t.Fatal("expected valid float")
			} else if nullFloat.Float64 != 3.14 {
				t.Fatalf("expected 3.14, got %v", nullFloat.Float64)
			}
		})

		t.Run("sql.NullInt32, NullInt16, NullByte", func(t *testing.T) {
			idCounter++
			nullID := idCounter
			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, nullable_int) VALUES (?, NULL)", nullID); err != nil {
				t.Fatal(err)
			}

			var nullInt32 sql.NullInt32
			if err := db.QueryRow("SELECT nullable_int FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullInt32); err != nil {
				t.Fatal(err)
			} else if nullInt32.Valid {
				t.Fatal("expected NULL for NullInt32")
			}

			var nullInt16 sql.NullInt16
			if err := db.QueryRow("SELECT nullable_int FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullInt16); err != nil {
				t.Fatal(err)
			} else if nullInt16.Valid {
				t.Fatal("expected NULL for NullInt16")
			}

			var nullByte sql.NullByte
			if err := db.QueryRow("SELECT nullable_int FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullByte); err != nil {
				t.Fatal(err)
			} else if nullByte.Valid {
				t.Fatal("expected NULL for NullByte")
			}

			idCounter++
			valID := idCounter
			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, nullable_int) VALUES (?, ?)", valID, 100); err != nil {
				t.Fatal(err)
			}

			if err := db.QueryRow("SELECT nullable_int FROM test_sqldatatypes WHERE id = ?", valID).Scan(&nullInt32); err != nil {
				t.Fatal(err)
			} else if !nullInt32.Valid {
				t.Fatal("expected valid NullInt32")
			} else if nullInt32.Int32 != 100 {
				t.Fatalf("expected 100, got %d", nullInt32.Int32)
			}
		})

		t.Run("sql.NullBool", func(t *testing.T) {
			idCounter++
			nullID := idCounter
			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, bool_val) VALUES (?, NULL)", nullID); err != nil {
				t.Fatal(err)
			}

			var nullBool sql.NullBool
			if err := db.QueryRow("SELECT bool_val FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullBool); err != nil {
				t.Fatal(err)
			} else if nullBool.Valid {
				t.Fatal("expected NULL for NullBool")
			}

			idCounter++
			trueID := idCounter
			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, bool_val) VALUES (?, ?)", trueID, 1); err != nil {
				t.Fatal(err)
			}

			if err := db.QueryRow("SELECT bool_val FROM test_sqldatatypes WHERE id = ?", trueID).Scan(&nullBool); err != nil {
				t.Fatal(err)
			} else if !nullBool.Valid {
				t.Fatal("expected valid NullBool")
			} else if !nullBool.Bool {
				t.Fatal("expected true")
			}
		})
	})

	t.Run("Time Types", func(t *testing.T) {
		tests := []struct {
			name  string
			value time.Time
		}{
			{"current time", time.Now().UTC().Truncate(time.Second)},
			{"unix epoch", time.Unix(0, 0).UTC()},
			{"far future", time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)},
			{"far past", time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				idCounter++
				currentID := idCounter

				formatted := tt.value.Format(time.RFC3339)

				if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, time_val) VALUES (?, ?)", currentID, formatted); err != nil {
					t.Fatal(err)
				}

				var timeStr string
				if err := db.QueryRow("SELECT time_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&timeStr); err != nil {
					t.Fatal(err)
				}

				if parsed, err := time.Parse(time.RFC3339, timeStr); err != nil {
					t.Fatal(err)
				} else if tt.value.Unix() != parsed.Unix() {
					t.Fatalf("time mismatch for %s: want %v, got %v", tt.name, tt.value.Unix(), parsed.Unix())
				}
			})
		}

		t.Run("NULL time with sql.NullTime", func(t *testing.T) {
			idCounter++
			nullID := idCounter
			if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, time_val) VALUES (?, NULL)", nullID); err != nil {
				t.Fatal(err)
			}

			var nullTime sql.NullTime
			if err := db.QueryRow("SELECT time_val FROM test_sqldatatypes WHERE id = ?", nullID).Scan(&nullTime); err != nil {
				t.Fatal(err)
			} else if nullTime.Valid {
				t.Fatal("expected NULL for NullTime")
			}
		})
	})

	t.Run("Overflow Detection", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int64
			scanType  string
			expectErr bool
		}{
			{"int64(128) into int8", 128, "int8", true},
			{"int64(256) into uint8", 256, "uint8", true},
			{"int64(-1) into uint", -1, "uint", true},
			{"int64(65536) into uint16", 65536, "uint16", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				idCounter++
				currentID := idCounter

				if _, err := db.Exec("INSERT INTO test_sqldatatypes (id, int_val) VALUES (?, ?)", currentID, tt.value); err != nil {
					t.Fatal(err)
				}

				var err2 error
				switch tt.scanType {
				case "int8":
					var v int8
					err2 = db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v)
				case "uint8":
					var v uint8
					err2 = db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v)
				case "uint":
					var v uint
					err2 = db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v)
				case "uint16":
					var v uint16
					err2 = db.QueryRow("SELECT int_val FROM test_sqldatatypes WHERE id = ?", currentID).Scan(&v)
				}

				if tt.expectErr {
					if err2 == nil {
						t.Fatalf("expected overflow error for %s", tt.name)
					}
				} else if err2 != nil {
					t.Fatal(err2)
				}
			})
		}
	})
}

// Test_RealColumnPrecision verifies that REAL and NUMERIC columns return correct
// float64 values for all rows, including whole-number values like 1.0 that
// rqlite encodes as "1" in JSON (no decimal point).
// NUMERIC affinity makes this harder: SQLite stores whole numbers as integers
// and fractional numbers as reals, so the driver must handle both from a single
// column type.
func Test_EndToEnd_RealColumnPrecision(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	values := []struct {
		id  int
		val float64
	}{
		{1, 1.0},            // whole number — JSON: "1"
		{2, 2.0},            // whole number — JSON: "2"
		{3, 1.5},            // fractional — JSON: "1.5"
		{4, math.Pi},        // irrational — JSON: "3.141592653589793"
		{5, 1e10},           // scientific — JSON: "10000000000"
		{6, 1.000000000001}, // near-integer — requires float64 precision
	}

	for _, colType := range []string{"REAL", "NUMERIC"} {
		t.Run(colType, func(t *testing.T) {
			table := "test_realcolumnprecision_" + strings.ToLower(colType)

			_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
			defer db.Exec("DROP TABLE IF EXISTS " + table)

			if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, val " + colType + ")"); err != nil {
				t.Fatal(err)
			}

			for _, r := range values {
				if _, err := db.Exec("INSERT INTO "+table+" (id, val) VALUES (?, ?)", r.id, r.val); err != nil {
					t.Fatal(err)
				}
			}

			rows, err := db.Query("SELECT id, val FROM " + table + " ORDER BY id")
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			var got []float64
			for rows.Next() {
				var id int
				var val float64
				if err := rows.Scan(&id, &val); err != nil {
					t.Fatal(err)
				}
				got = append(got, val)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(values) {
				t.Fatalf("expected %d rows, got %d", len(values), len(got))
			}

			for i, r := range values {
				if math.Float64bits(r.val) != math.Float64bits(got[i]) {
					t.Fatalf("row %d (val=%v): precision lost: want bits %016x, got %016x", r.id, r.val, math.Float64bits(r.val), math.Float64bits(got[i]))
				}
			}
		})
	}
}

// Test_NamedParameters verifies that named parameters are resolved by name, not
// by the order they are passed. The SQL references :b before :a, but the args
// supply "a" first — the values must still land in the correct columns.
func Test_EndToEnd_NamedParameters(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_namedparameters")
	defer db.Exec("DROP TABLE IF EXISTS test_namedparameters")

	if _, err := db.Exec("CREATE TABLE test_namedparameters (a TEXT, b TEXT)"); err != nil {
		t.Fatal(err)
	}

	// :b appears before :a in the SQL, but "a" is passed first in the args.
	if _, err := db.Exec("INSERT INTO test_namedparameters (a, b) VALUES (:b, :a)",
		sql.Named("a", "value_a"),
		sql.Named("b", "value_b"),
	); err != nil {
		t.Fatal(err)
	}

	var a, b string
	if err := db.QueryRow("SELECT a, b FROM test_namedparameters").Scan(&a, &b); err != nil {
		t.Fatal(err)
	} else if a != "value_b" {
		// :b appears first in the SQL (goes into column a) and :a appears second
		// (goes into column b), so the values must be swapped relative to arg order.
		t.Fatalf("expected a=%q, got %q", "value_b", a)
	} else if b != "value_a" {
		t.Fatalf("expected b=%q, got %q", "value_a", b)
	}
}

func Test_EndToEnd_TxCommit(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_txcommit")
	defer db.Exec("DROP TABLE IF EXISTS test_txcommit")

	if _, err := db.Exec("CREATE TABLE test_txcommit (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO test_txcommit (id, name) VALUES (1, 'alice')"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO test_txcommit (id, name) VALUES (2, 'bob')"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	ids := collectIDs(t, db, "test_txcommit")
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("expected ids [1 2], got %v", ids)
	}
}

func Test_EndToEnd_TxRollback(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_txrollback")
	defer db.Exec("DROP TABLE IF EXISTS test_txrollback")

	if _, err := db.Exec("CREATE TABLE test_txrollback (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO test_txrollback (id, name) VALUES (1, 'alice')"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	ids := collectIDs(t, db, "test_txrollback")
	if len(ids) != 0 {
		t.Fatalf("expected no rows after rollback, got %v", ids)
	}
}

func Test_EndToEnd_TxCommitWithError(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_txcommiterror")
	defer db.Exec("DROP TABLE IF EXISTS test_txcommiterror")

	if _, err := db.Exec("CREATE TABLE test_txcommiterror (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO test_txcommiterror (id, name) VALUES (1, 'alice')"); err != nil {
		t.Fatal(err)
	}
	// Duplicate PK — will fail at commit time
	if _, err := tx.Exec("INSERT INTO test_txcommiterror (id, name) VALUES (1, 'duplicate')"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); !errors.Is(err, stdlib.ErrQuery) {
		t.Fatalf("expected ErrQuery from Commit, got: %v", err)
	}
	ids := collectIDs(t, db, "test_txcommiterror")
	if len(ids) != 0 {
		t.Fatalf("expected no committed rows, got %v", ids)
	}
}

func Test_EndToEnd_QueryInTransactionBlocked(t *testing.T) {
	db := testDatabase(t, "rqlite")
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if _, err := tx.Query("SELECT 1"); !errors.Is(err, stdlib.ErrQuery) {
		t.Fatalf("expected ErrQuery when querying inside transaction without AllowQueryInTxn, got: %v", err)
	}
}

func Test_EndToEnd_QueryInTransactionAllowed(t *testing.T) {
	db := testDatabase(t, "rqlite-allowquery")
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT 1")
	if err != nil {
		t.Fatalf("expected query to succeed with AllowQueryInTxn, got: %v", err)
	}
	rows.Close()
}

func collectIDs(t *testing.T, db *sql.DB, table string) []int64 {
	t.Helper()
	rows, err := db.Query("SELECT id FROM " + table + " ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

// Test_MultiStatementExec_LastInsertID verifies that the last insert id from a
// multi-statement exec reflects the LAST statement, both when sent via the
// database/sql driver (semicolon-separated SQL) and via rqlitehttp directly
// (separate SQLStatements in one request).
func Test_EndToEnd_MultiStatementExec_LastInsertID(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	ctx := context.Background()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var client *rqlitehttp.Client
	if err := conn.Raw(func(c any) error {
		stdlibConn, ok := c.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("unexpected connection type: %T", c)
		}
		client = stdlibConn.Client
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	const table = "test_multistatementexec_lastinsertid"

	t.Run("database/sql driver", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TABLE IF EXISTS " + table)

		result, err := db.ExecContext(ctx,
			"INSERT INTO "+table+" (name) VALUES ('first'); "+
				"INSERT INTO "+table+" (name) VALUES ('second'); "+
				"INSERT INTO "+table+" (name) VALUES ('third')")
		if err != nil {
			t.Fatal(err)
		}

		if lastID, err := result.LastInsertId(); err != nil {
			t.Fatal(err)
		} else if lastID != 3 {
			t.Fatalf("LastInsertId should be the id of the third insert, not the first: want 3, got %d", lastID)
		}
	})

	t.Run("rqlitehttp direct", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TABLE IF EXISTS " + table)

		resp, err := client.Execute(ctx, rqlitehttp.SQLStatements{
			{SQL: "INSERT INTO " + table + " (name) VALUES ('first')"},
			{SQL: "INSERT INTO " + table + " (name) VALUES ('second')"},
			{SQL: "INSERT INTO " + table + " (name) VALUES ('third')"},
		}, &rqlitehttp.ExecuteOptions{})
		if err != nil {
			t.Fatal(err)
		} else if resp.Error != "" {
			t.Fatalf("unexpected request error: %s", resp.Error)
		} else if len(resp.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(resp.Results))
		}

		for i, r := range resp.Results {
			if r.Error != "" {
				t.Fatalf("statement %d should succeed: got error %s", i, r.Error)
			}
		}

		lastID := resp.Results[len(resp.Results)-1].LastInsertID
		if lastID != 3 {
			t.Fatalf("last insert id should be the id of the third insert, not the first: want 3, got %d", lastID)
		} else if resp.Results[0].LastInsertID == lastID {
			t.Fatal("first and last insert ids should differ")
		}
	})

	t.Run("rqlitehttp direct semicolon-separated", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TABLE IF EXISTS " + table)

		resp, err := client.Execute(ctx, rqlitehttp.SQLStatements{
			{SQL: "INSERT INTO " + table + " (name) VALUES ('first'); " +
				"INSERT INTO " + table + " (name) VALUES ('second'); " +
				"INSERT INTO " + table + " (name) VALUES ('third')"},
		}, &rqlitehttp.ExecuteOptions{})
		if err != nil {
			t.Fatal(err)
		} else if resp.Error != "" {
			t.Fatalf("unexpected request error: %s", resp.Error)
		} else if len(resp.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(resp.Results))
		} else if resp.Results[0].Error != "" {
			t.Fatalf("unexpected result error: %s", resp.Results[0].Error)
		}

		lastID := resp.Results[len(resp.Results)-1].LastInsertID
		if lastID != 3 {
			t.Fatalf("last insert id should be the id of the third insert, not the first: want 3, got %d", lastID)
		}
	})
}

// Test_MultiStatementExec_PartialFailure verifies that when a middle statement
// fails in a multi-statement exec, the surrounding statements still commit
// (no implicit transaction). Behavior is compared between the database/sql
// driver (semicolon-separated SQL) and rqlitehttp directly (separate
// SQLStatements).
func Test_EndToEnd_MultiStatementExec_PartialFailure(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	ctx := context.Background()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var client *rqlitehttp.Client
	if err := conn.Raw(func(c any) error {
		client = c.(*stdlib.Conn).Client
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Three statements: INSERT id=1 (ok), INSERT id=1 again (duplicate PK → fail),
	// INSERT id=2 (ok). With no implicit transaction the first and third should commit.
	const table = "test_multistatementexec_partialfailure"

	t.Run("database/sql driver", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TABLE IF EXISTS " + table)

		// Semicolon-joined SQL is sent as a single SQLStatement to rqlite.
		// rqlite returns a single Result whose Error field holds the first failure,
		// so the driver does surface the error.
		if _, err := db.ExecContext(ctx,
			"INSERT INTO "+table+" VALUES (1, 'first'); "+
				"INSERT INTO "+table+" VALUES (1, 'duplicate'); "+
				"INSERT INTO "+table+" VALUES (2, 'third')"); err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("expected error containing %q, got: %v", "UNIQUE constraint failed", err)
		}

		// rqlite stops executing the remaining statements after the first error in a
		// single SQL string, so id=2 never runs. Only id=1 (before the error) commits.
		// This differs from the rqlitehttp direct path below where each SQLStatement
		// is independent and the third statement still commits.
		ids := collectIDs(t, db, table)
		if len(ids) != 1 || ids[0] != 1 {
			t.Fatalf("only id=1 should exist: execution stops at the first error in a single SQL string: got %v", ids)
		}
	})

	t.Run("rqlitehttp direct", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TABLE IF EXISTS " + table)

		resp, err := client.Execute(ctx, rqlitehttp.SQLStatements{
			{SQL: "INSERT INTO " + table + " VALUES (1, 'first')"},
			{SQL: "INSERT INTO " + table + " VALUES (1, 'duplicate')"},
			{SQL: "INSERT INTO " + table + " VALUES (2, 'third')"},
		}, &rqlitehttp.ExecuteOptions{})
		if err != nil {
			t.Fatal(err)
		} else if resp.Error != "" {
			t.Fatalf("unexpected request error: %s", resp.Error)
		} else if len(resp.Results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(resp.Results))
		}

		if resp.Results[0].Error != "" {
			t.Fatalf("first statement should succeed: got error %s", resp.Results[0].Error)
		} else if resp.Results[1].Error == "" {
			t.Fatal("second statement should fail (duplicate PK)")
		} else if resp.Results[2].Error != "" {
			t.Fatalf("third statement should succeed despite second failing: got error %s", resp.Results[2].Error)
		}

		ids := collectIDs(t, db, table)
		if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
			t.Fatalf("rows 1 and 2 should exist; duplicate insert should be the only one absent: got %v", ids)
		}
	})

	t.Run("rqlitehttp direct semicolon-separated", func(t *testing.T) {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
		if _, err := db.Exec("CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
			t.Fatal(err)
		}
		defer db.Exec("DROP TABLE IF EXISTS " + table)

		resp, err := client.Execute(ctx, rqlitehttp.SQLStatements{
			{SQL: "INSERT INTO " + table + " VALUES (1, 'first'); " +
				"INSERT INTO " + table + " VALUES (1, 'duplicate'); " +
				"INSERT INTO " + table + " VALUES (2, 'third')"},
		}, &rqlitehttp.ExecuteOptions{})
		if err != nil {
			t.Fatal(err)
		} else if resp.Error != "" {
			t.Fatalf("unexpected request error: %s", resp.Error)
		} else if len(resp.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(resp.Results))
		} else if !strings.Contains(resp.Results[0].Error, "UNIQUE constraint failed") {
			t.Fatalf("single-result error should match driver behaviour: got %q", resp.Results[0].Error)
		}

		ids := collectIDs(t, db, table)
		if len(ids) != 1 || ids[0] != 1 {
			t.Fatalf("only id=1 should exist: same stop-on-error behaviour as the driver path: got %v", ids)
		}
	})
}

func Test_EndToEnd_JSONNumber(t *testing.T) {
	db := testDatabase(t)
	defer db.Close()

	_, _ = db.Exec("DROP TABLE IF EXISTS test_jsonnumber")
	defer db.Exec("DROP TABLE IF EXISTS test_jsonnumber")

	if _, err := db.Exec("CREATE TABLE test_jsonnumber (i INTEGER, r REAL, n NUMERIC)"); err != nil {
		t.Fatal(err)
	}

	iVal := json.Number("42")
	rVal := json.Number("3.14")
	nIntVal := json.Number("100")
	nFracVal := json.Number("1.5")

	if _, err := db.Exec("INSERT INTO test_jsonnumber (i, r, n) VALUES (?, ?, ?)", iVal, rVal, nIntVal); err != nil {
		t.Fatalf("insert integer/real/numeric(int): %s", err)
	}
	if _, err := db.Exec("INSERT INTO test_jsonnumber (i, r, n) VALUES (?, ?, ?)", iVal, rVal, nFracVal); err != nil {
		t.Fatalf("insert integer/real/numeric(frac): %s", err)
	}

	rows, err := db.Query("SELECT i, r, n FROM test_jsonnumber ORDER BY n")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type row struct{ i, r, n json.Number }
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.i, &r.r, &r.n); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}

	// Rows ordered by n ASC: nFracVal (1.5) first, nIntVal (100) second.
	checks := []struct {
		row   row
		wantI json.Number
		wantR json.Number
		wantN json.Number
	}{
		{got[0], iVal, rVal, nFracVal},
		{got[1], iVal, rVal, nIntVal},
	}
	for _, c := range checks {
		if c.row.i != c.wantI {
			t.Errorf("i: got %q, want %q", c.row.i, c.wantI)
		}
		if c.row.r != c.wantR {
			t.Errorf("r: got %q, want %q", c.row.r, c.wantR)
		}
		if c.row.n != c.wantN {
			t.Errorf("n: got %q, want %q", c.row.n, c.wantN)
		}
	}
}
