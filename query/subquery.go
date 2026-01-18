package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

// Subquery represents a SELECT query used as a subquery.
type Subquery struct {
	builder *SelectBuilder
	alias   string
}

// AsSubquery converts a SelectBuilder into a Subquery with an alias.
func (s *SelectBuilder) AsSubquery(alias string) *Subquery {
	return &Subquery{
		builder: s,
		alias:   alias,
	}
}

// inSubqueryCondition represents a column IN (subquery) condition.
type inSubqueryCondition struct {
	column   string
	subquery *SelectBuilder
	negate   bool
}

// InSubquery creates a condition for column IN (SELECT ...).
// Errors during subquery building are handled gracefully in ToSQL() instead of panicking.
func InSubquery(column string, subquery *SelectBuilder) Condition {
	return &inSubqueryCondition{
		column:   column,
		subquery: subquery,
		negate:   false,
	}
}

// NotInSubquery creates a condition for column NOT IN (SELECT ...).
// Errors during subquery building are handled gracefully in ToSQL() instead of panicking.
func NotInSubquery(column string, subquery *SelectBuilder) Condition {
	return &inSubqueryCondition{
		column:   column,
		subquery: subquery,
		negate:   true,
	}
}

func (c *inSubqueryCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	subSQL, subArgs, err := c.subquery.Build()
	if err != nil {
		// Return an error SQL fragment instead of panicking.
		// This gracefully degrades instead of crashing the application.
		return fmt.Sprintf("/* ERROR: InSubquery build failed for column %q: %v */ 1=0",
			c.column, err), nil, paramStart
	}
	renumberedSQL := renumberParameters(dialect, subSQL, len(subArgs), paramStart)

	operator := "IN"
	if c.negate {
		operator = "NOT IN"
	}

	return fmt.Sprintf("%s %s (%s)",
			dialect.QuoteIdentifier(c.column),
			operator,
			renumberedSQL),
		subArgs,
		paramStart + len(subArgs)
}

// existsCondition represents an EXISTS (subquery) condition.
type existsCondition struct {
	subquery *SelectBuilder
	negate   bool
}

// Exists creates a condition for EXISTS (SELECT ...).
// Errors during subquery building are handled gracefully in ToSQL() instead of panicking.
func Exists(subquery *SelectBuilder) Condition {
	return &existsCondition{subquery: subquery, negate: false}
}

// NotExists creates a condition for NOT EXISTS (SELECT ...).
// Errors during subquery building are handled gracefully in ToSQL() instead of panicking.
func NotExists(subquery *SelectBuilder) Condition {
	return &existsCondition{subquery: subquery, negate: true}
}

func (c *existsCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	subSQL, subArgs, err := c.subquery.Build()
	if err != nil {
		// Return an error SQL fragment instead of panicking.
		// This gracefully degrades instead of crashing the application.
		operator := "EXISTS"
		if c.negate {
			operator = "NOT EXISTS"
		}
		return fmt.Sprintf("/* ERROR: %s build failed: %v */ 1=0", operator, err), nil, paramStart
	}
	renumberedSQL := renumberParameters(dialect, subSQL, len(subArgs), paramStart)

	operator := "EXISTS"
	if c.negate {
		operator = "NOT EXISTS"
	}

	return fmt.Sprintf("%s (%s)", operator, renumberedSQL), subArgs, paramStart + len(subArgs)
}

// renumberParameters renumbers parameter placeholders in a SQL query.
// This is used when embedding subqueries to ensure parameters are numbered correctly.
func renumberParameters(dialect database.Dialect, sql string, argCount int, startFrom int) string {
	result := sql

	// Replace placeholders from highest to lowest to avoid double replacement
	for i := argCount; i >= 1; i-- {
		oldPlaceholder := dialect.Placeholder(i)
		newPlaceholder := dialect.Placeholder(startFrom + i - 1)
		result = strings.Replace(result, oldPlaceholder, newPlaceholder, 1)
	}

	return result
}
