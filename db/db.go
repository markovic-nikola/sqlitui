package db

import (
	"database/sql"
	"fmt"
	"strings"

	// Import the CGo-free SQLite driver. The underscore means we import
	// it only for its side effect: registering itself as a database/sql
	// driver under the name "sqlite".
	_ "modernc.org/sqlite"
)

// Open connects to a SQLite database file. It uses the standard
// database/sql interface, so all the usual Query/Exec methods work.
func Open(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

// ListTables returns the names of all user-created tables in the database.
// sqlite_master is a system table that stores the schema — every CREATE TABLE
// statement lives here as a row with type='table'.
func ListTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(
		"SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// GetColumns returns column names for a table using PRAGMA table_info.
// This is a SQLite-specific command that returns schema metadata.
func GetColumns(db *sql.DB, table string) ([]string, error) {
	rows, err := db.Query("PRAGMA table_info(" + quoteIdent(table) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

// GetRows fetches up to `limit` rows from a table, returning rowids and all
// values as strings. The rowid is selected separately so DELETE/UPDATE can
// target the exact row regardless of primary key shape.
func GetRows(db *sql.DB, table string, limit, offset int) ([]string, []int64, [][]string, error) {
	rows, err := db.Query("SELECT rowid, * FROM "+quoteIdent(table)+" LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	return scanRowsWithRowID(rows)
}

// ExecQuery runs an arbitrary SQL query and returns columns + string rows.
// Intended for custom queries from the query popup.
func ExecQuery(db *sql.DB, query string) ([]string, [][]string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// FilterColumn searches a table for rows where a single column matches the
// query (case-insensitive LIKE). Single-column search is fast even on large tables.
func FilterColumn(db *sql.DB, table, column, query string, limit, offset int) ([]string, []int64, [][]string, error) {
	q := "SELECT rowid, * FROM " + quoteIdent(table) + " WHERE " + quoteIdent(column) + " LIKE ? COLLATE NOCASE LIMIT ? OFFSET ?"
	rows, err := db.Query(q, "%"+query+"%", limit, offset)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	return scanRowsWithRowID(rows)
}

// DeleteRow removes a single row from a table identified by its rowid.
// Works for any default SQLite table (i.e., not declared WITHOUT ROWID).
func DeleteRow(db *sql.DB, table string, rowid int64) error {
	_, err := db.Exec("DELETE FROM "+quoteIdent(table)+" WHERE rowid = ?", rowid)
	return err
}

// CountRows returns the total number of rows in a table.
func CountRows(db *sql.DB, table string) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + quoteIdent(table)).Scan(&count)
	return count, err
}

// CountFilteredRows returns the number of rows matching a LIKE filter.
func CountFilteredRows(db *sql.DB, table, column, query string) (int, error) {
	var count int
	q := "SELECT COUNT(*) FROM " + quoteIdent(table) + " WHERE " + quoteIdent(column) + " LIKE ? COLLATE NOCASE"
	err := db.QueryRow(q, "%"+query+"%").Scan(&count)
	return count, err
}

// scanRowsWithRowID expects the first selected column to be `rowid`. It splits
// rowids out into their own slice and returns the remaining columns as strings.
func scanRowsWithRowID(rows *sql.Rows) ([]string, []int64, [][]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, nil, err
	}
	if len(cols) == 0 {
		return cols, nil, nil, nil
	}
	userCols := cols[1:]

	var rowids []int64
	var result [][]string
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, nil, err
		}
		var rid int64
		switch v := values[0].(type) {
		case int64:
			rid = v
		case int:
			rid = int64(v)
		}
		rowids = append(rowids, rid)
		row := make([]string, len(userCols))
		for i, v := range values[1:] {
			if v == nil {
				row[i] = "NULL"
			} else if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result = append(result, row)
	}
	return userCols, rowids, result, rows.Err()
}

// scanRows reads all rows from a *sql.Rows result set, returning column
// names and all values as strings. Used by ExecQuery for arbitrary user queries.
func scanRows(rows *sql.Rows) ([]string, [][]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var result [][]string
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				row[i] = "NULL"
			} else if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result = append(result, row)
	}
	return cols, result, rows.Err()
}

// quoteIdent wraps a table/column name in double quotes to prevent SQL injection.
// Any embedded double quotes are doubled (standard SQL escaping).
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
