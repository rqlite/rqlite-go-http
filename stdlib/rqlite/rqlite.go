package rqlite

import (
	"database/sql"

	"github.com/rqlite/rqlite-go-http/stdlib"
)

func init() {
	sql.Register("rqlite", &stdlib.Driver{})
}
