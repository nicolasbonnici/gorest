package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

// UpdateBuilder builds UPDATE queries with support for WHERE conditions and RETURNING clauses.
type UpdateBuilder struct {
	dialect    database.Dialect
	table      string
	sets       map[string]any
	setOrder   []string
	conditions []Condition
	returning  []string
	err        error
}

// Set adds or updates a column value in the SET clause.
// Multiple calls to Set will add multiple columns to update.
func (u *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	if _, exists := u.sets[column]; !exists {
		u.setOrder = append(u.setOrder, column)
	}
	u.sets[column] = value
	return u
}

// SetMap adds multiple column updates from a map.
// This is a convenience method for setting multiple columns at once.
func (u *UpdateBuilder) SetMap(m map[string]any) *UpdateBuilder {
	for col, val := range m {
		u.Set(col, val)
	}
	return u
}

// Where adds a WHERE condition to the UPDATE statement.
// Multiple calls to Where will combine conditions with AND.
func (u *UpdateBuilder) Where(condition Condition) *UpdateBuilder {
	u.conditions = append(u.conditions, condition)
	return u
}

// And is an alias for Where that adds an AND condition.
func (u *UpdateBuilder) And(condition Condition) *UpdateBuilder {
	return u.Where(condition)
}

// Or combines the previous condition with the given condition using OR.
// If no previous condition exists, this behaves like Where.
func (u *UpdateBuilder) Or(condition Condition) *UpdateBuilder {
	if len(u.conditions) == 0 {
		return u.Where(condition)
	}
	last := u.conditions[len(u.conditions)-1]
	u.conditions[len(u.conditions)-1] = Or(last, condition)
	return u
}

// Returning specifies columns to return after the update (PostgreSQL, SQLite).
// This is ignored for databases that don't support RETURNING.
func (u *UpdateBuilder) Returning(columns ...string) *UpdateBuilder {
	u.returning = columns
	return u
}

// Build generates the final SQL query and arguments.
func (u *UpdateBuilder) Build() (query string, args []any, err error) {
	// Return early if there was a validation error
	if u.err != nil {
		return "", nil, u.err
	}

	// Validate table name
	if err := ValidateIdentifier(u.table); err != nil {
		return "", nil, err
	}

	var parts []string
	var allArgs []any
	paramIdx := 1

	parts = append(parts, "UPDATE", u.dialect.QuoteIdentifier(u.table))

	var setClauses []string
	for _, col := range u.setOrder {
		val := u.sets[col]
		setClauses = append(setClauses, fmt.Sprintf("%s = %s",
			u.dialect.QuoteIdentifier(col),
			u.dialect.Placeholder(paramIdx)))
		allArgs = append(allArgs, val)
		paramIdx++
	}
	parts = append(parts, "SET", strings.Join(setClauses, ", "))

	if len(u.conditions) > 0 {
		whereSQL, whereArgs, _ := And(u.conditions...).ToSQL(u.dialect, paramIdx)
		parts = append(parts, "WHERE", whereSQL)
		allArgs = append(allArgs, whereArgs...)
	}

	if len(u.returning) > 0 && u.dialect.SupportsReturning() {
		parts = append(parts, u.dialect.ReturningClause(u.returning...))
	}

	return strings.Join(parts, " "), allArgs, nil
}
