package sqlitemod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "sqlite",
		Type: "SQLite",
		Doc:  "SQLite database access via Go's database/sql.",
		Funcs: []modules.FuncDef{
			{Name: "open", Args: []modules.ArgType{modules.String}, Doc: "Open a SQLite database file. Use \":memory:\" for in-memory databases. Returns a connection handle."},
			{Name: "exec", Args: []modules.ArgType{modules.Any, modules.String}, Variadic: true, Doc: "Execute a SQL statement (DDL/DML). Returns the number of rows affected."},
			{Name: "query", Args: []modules.ArgType{modules.Any, modules.String}, Variadic: true, Doc: "Execute a SQL query. Returns an array of hashes."},
			{Name: "query_row", Args: []modules.ArgType{modules.Any, modules.String}, Variadic: true, Doc: "Execute a SQL query. Returns the first row as a hash, or nil if no rows."},
			{Name: "query_val", Args: []modules.ArgType{modules.Any, modules.String}, Variadic: true, Doc: "Execute a SQL query. Returns the first column of the first row, or nil if no rows."},
			{Name: "close", Args: []modules.ArgType{modules.Any}, Doc: "Close a database connection."},
		},
		GoImports: []string{"database/sql", "math", `_ "modernc.org/sqlite"`},
		GoDeps:    []string{"modernc.org/sqlite v1.44.3"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
