//go:build ignore

package sqlitemod

import (
	"database/sql"
	"fmt"
	"math"

	_ "modernc.org/sqlite"
)

// --- sqlite module ---

type SQLite struct{}

// Open opens a SQLite database and returns the connection handle.
func (*SQLite) Open(path string) interface{} {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		panic(fmt.Sprintf("sqlite.open: %v", err))
	}
	if err := db.Ping(); err != nil {
		db.Close()
		panic(fmt.Sprintf("sqlite.open: %v", err))
	}
	return db
}

// Exec executes a SQL statement and returns the number of rows affected.
func (*SQLite) Exec(conn interface{}, query string, params ...interface{}) interface{} {
	db := assertConn(conn, "exec")
	args := toDriverArgs(params)
	result, err := db.Exec(query, args...)
	if err != nil {
		panic(fmt.Sprintf("sqlite.exec: %v", err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0
	}
	return int(n)
}

// Query executes a SQL query and returns all rows as an array of hashes.
func (*SQLite) Query(conn interface{}, query string, params ...interface{}) interface{} {
	db := assertConn(conn, "query")
	args := toDriverArgs(params)
	rows, err := db.Query(query, args...)
	if err != nil {
		panic(fmt.Sprintf("sqlite.query: %v", err))
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		panic(fmt.Sprintf("sqlite.query: %v", err))
	}

	var result []interface{}
	for rows.Next() {
		row := scanRow(cols, rows)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Sprintf("sqlite.query: %v", err))
	}
	if result == nil {
		return []interface{}{}
	}
	return result
}

// QueryRow executes a SQL query and returns the first row as a hash, or nil.
func (*SQLite) QueryRow(conn interface{}, query string, params ...interface{}) interface{} {
	db := assertConn(conn, "query_row")
	args := toDriverArgs(params)
	rows, err := db.Query(query, args...)
	if err != nil {
		panic(fmt.Sprintf("sqlite.query_row: %v", err))
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		panic(fmt.Sprintf("sqlite.query_row: %v", err))
	}

	if !rows.Next() {
		return nil
	}
	return scanRow(cols, rows)
}

// QueryVal executes a SQL query and returns the first column of the first row, or nil.
func (*SQLite) QueryVal(conn interface{}, query string, params ...interface{}) interface{} {
	db := assertConn(conn, "query_val")
	args := toDriverArgs(params)
	rows, err := db.Query(query, args...)
	if err != nil {
		panic(fmt.Sprintf("sqlite.query_val: %v", err))
	}
	defer rows.Close()

	if !rows.Next() {
		return nil
	}

	var val interface{}
	if err := rows.Scan(&val); err != nil {
		panic(fmt.Sprintf("sqlite.query_val: %v", err))
	}
	return normalizeValue(val)
}

// Close closes a database connection.
func (*SQLite) Close(conn interface{}) interface{} {
	db := assertConn(conn, "close")
	if err := db.Close(); err != nil {
		panic(fmt.Sprintf("sqlite.close: %v", err))
	}
	return nil
}

// --- internal helpers ---

// assertConn type-asserts the connection handle or panics with a friendly message.
func assertConn(conn interface{}, funcName string) *sql.DB {
	db, ok := conn.(*sql.DB)
	if !ok {
		panic(fmt.Sprintf("sqlite.%s: expected a connection from sqlite.open, got %T", funcName, conn))
	}
	return db
}

// toDriverArgs converts Rugo values to driver-friendly values.
func toDriverArgs(params []interface{}) []interface{} {
	args := make([]interface{}, len(params))
	for i, p := range params {
		switch v := p.(type) {
		case bool:
			if v {
				args[i] = 1
			} else {
				args[i] = 0
			}
		default:
			args[i] = p
		}
	}
	return args
}

// scanRow scans a single row into a Rugo hash.
func scanRow(cols []string, rows *sql.Rows) interface{} {
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		panic(fmt.Sprintf("sqlite: scan error: %v", err))
	}
	row := make(map[interface{}]interface{}, len(cols))
	for i, col := range cols {
		row[col] = normalizeValue(vals[i])
	}
	return row
}

// normalizeValue converts database values to Rugo-friendly types.
func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case nil:
		return nil
	case int64:
		if val >= math.MinInt && val <= math.MaxInt {
			return int(val)
		}
		return val
	case float64:
		return val
	case []byte:
		return string(val)
	case string:
		return val
	case bool:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Silence unused import warnings.
var _ = math.MaxInt
