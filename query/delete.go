package query

import (
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

// DeleteBuilder builds DELETE queries with support for WHERE conditions and RETURNING clauses.
type DeleteBuilder struct {
	dialect    database.Dialect
	table      string
	conditions []Condition
	returning  []string
	err        error
}

// Where adds a WHERE condition to the DELETE statement.
// Multiple calls to Where will combine conditions with AND.
func (d *DeleteBuilder) Where(condition Condition) *DeleteBuilder {
	d.conditions = append(d.conditions, condition)
	return d
}

// And is an alias for Where that adds an AND condition.
func (d *DeleteBuilder) And(condition Condition) *DeleteBuilder {
	return d.Where(condition)
}

// Or combines the previous condition with the given condition using OR.
// If no previous condition exists, this behaves like Where.
func (d *DeleteBuilder) Or(condition Condition) *DeleteBuilder {
	if len(d.conditions) == 0 {
		return d.Where(condition)
	}
	last := d.conditions[len(d.conditions)-1]
	d.conditions[len(d.conditions)-1] = Or(last, condition)
	return d
}

// Returning specifies columns to return after the delete (PostgreSQL, SQLite).
// This is ignored for databases that don't support RETURNING.
func (d *DeleteBuilder) Returning(columns ...string) *DeleteBuilder {
	d.returning = columns
	return d
}

// Build generates the final SQL query and arguments.
func (d *DeleteBuilder) Build() (query string, args []any, err error) {
	// Return early if there was a validation error
	if d.err != nil {
		return "", nil, d.err
	}

	// Validate table name
	if err := ValidateIdentifier(d.table); err != nil {
		return "", nil, err
	}

	var parts []string
	var allArgs []any

	parts = append(parts, "DELETE FROM", d.dialect.QuoteIdentifier(d.table))

	if len(d.conditions) > 0 {
		whereSQL, whereArgs, _ := And(d.conditions...).ToSQL(d.dialect, 1)
		parts = append(parts, "WHERE", whereSQL)
		allArgs = append(allArgs, whereArgs...)
	}

	if len(d.returning) > 0 && d.dialect.SupportsReturning() {
		parts = append(parts, d.dialect.ReturningClause(d.returning...))
	}

	return strings.Join(parts, " "), allArgs, nil
}
