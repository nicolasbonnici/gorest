package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

// InsertBuilder builds INSERT queries with support for batch inserts and RETURNING clauses.
type InsertBuilder struct {
	dialect         database.Dialect
	table           string
	columns         []string
	values          [][]any
	returning       []string
	conflictColumns []string
	conflictAction  string
	err             error
}

// OnConflictDoNothing silently ignores rows that violate a unique constraint.
// Provide the conflict columns (e.g. the primary key columns).
// On MySQL, this emits INSERT IGNORE INTO; on PostgreSQL/SQLite it emits
// ON CONFLICT (cols) DO NOTHING.
func (i *InsertBuilder) OnConflictDoNothing(columns ...string) *InsertBuilder {
	i.conflictColumns = columns
	i.conflictAction = "DO NOTHING"
	return i
}

// Columns specifies the columns for the INSERT statement.
// This is optional when using ValuesMap, but required when using Values.
func (i *InsertBuilder) Columns(columns ...string) *InsertBuilder {
	i.columns = columns
	return i
}

// Values adds a row of values to insert.
// Multiple calls to Values will create a batch insert with multiple rows.
func (i *InsertBuilder) Values(values ...any) *InsertBuilder {
	i.values = append(i.values, values)
	return i
}

// ValuesMap adds a row from a map of column names to values.
// If columns haven't been set, they will be extracted from the map keys.
// Multiple calls to ValuesMap will create a batch insert with multiple rows.
func (i *InsertBuilder) ValuesMap(m map[string]any) *InsertBuilder {
	if len(i.columns) == 0 {
		for col := range m {
			i.columns = append(i.columns, col)
		}
	}

	vals := make([]any, len(i.columns))
	for idx, col := range i.columns {
		vals[idx] = m[col]
	}
	i.values = append(i.values, vals)
	return i
}

// Returning specifies columns to return after the insert (PostgreSQL, SQLite).
// This is ignored for databases that don't support RETURNING.
func (i *InsertBuilder) Returning(columns ...string) *InsertBuilder {
	i.returning = columns
	return i
}

// Build generates the final SQL query and arguments.
func (i *InsertBuilder) Build() (query string, args []any, err error) {
	// Return early if there was a validation error
	if i.err != nil {
		return "", nil, i.err
	}

	// Validate table name
	if err := ValidateIdentifier(i.table); err != nil {
		return "", nil, err
	}

	var parts []string
	var allArgs []any

	// MySQL uses INSERT IGNORE for "do nothing on conflict"; other dialects
	// handle it with an ON CONFLICT clause appended after VALUES.
	insertKeyword := "INSERT INTO"
	conflictClause := ""
	if i.conflictAction == "DO NOTHING" && len(i.conflictColumns) > 0 {
		conflictClause = i.dialect.OnConflictClause(i.conflictColumns, i.conflictAction)
		if conflictClause == "" {
			insertKeyword = "INSERT IGNORE INTO"
		}
	}

	parts = append(parts, insertKeyword, i.dialect.QuoteIdentifier(i.table))

	// Validate column names
	for _, col := range i.columns {
		if err := ValidateIdentifier(col); err != nil {
			return "", nil, fmt.Errorf("INSERT column validation: %w", err)
		}
	}

	quotedCols := make([]string, len(i.columns))
	for idx, col := range i.columns {
		quotedCols[idx] = i.dialect.QuoteIdentifier(col)
	}
	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))

	var valueClauses []string
	paramIdx := 1
	for _, row := range i.values {
		var placeholders []string
		for range row {
			placeholders = append(placeholders, i.dialect.Placeholder(paramIdx))
			paramIdx++
		}
		valueClauses = append(valueClauses, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
		allArgs = append(allArgs, row...)
	}
	parts = append(parts, "VALUES", strings.Join(valueClauses, ", "))

	if conflictClause != "" {
		parts = append(parts, conflictClause)
	}

	if len(i.returning) > 0 && i.dialect.SupportsReturning() {
		parts = append(parts, i.dialect.ReturningClause(i.returning...))
	}

	return strings.Join(parts, " "), allArgs, nil
}
