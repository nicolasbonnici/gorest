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

// errorCondition represents a condition that failed validation.
// It captures errors early and propagates them through the query building process.
type errorCondition struct {
	err error
}

func (c *errorCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	return fmt.Sprintf("/* ERROR: %v */ 1=0", c.err), nil, paramStart
}

// inSubqueryCondition represents a column IN (subquery) condition.
type inSubqueryCondition struct {
	column   string
	subquery *SelectBuilder
	negate   bool
}

// InSubquery creates a condition for column IN (SELECT ...).
// If the subquery fails to build, it returns an errorCondition instead of panicking.
func InSubquery(column string, subquery *SelectBuilder) Condition {
	if _, _, err := subquery.Build(); err != nil {
		return &errorCondition{
			err: fmt.Errorf("invalid subquery for column %q: %w", column, err),
		}
	}

	return &inSubqueryCondition{
		column:   column,
		subquery: subquery,
		negate:   false,
	}
}

// NotInSubquery creates a condition for column NOT IN (SELECT ...).
// If the subquery fails to build, it returns an errorCondition instead of panicking.
func NotInSubquery(column string, subquery *SelectBuilder) Condition {
	if _, _, err := subquery.Build(); err != nil {
		return &errorCondition{
			err: fmt.Errorf("invalid subquery for column %q: %w", column, err),
		}
	}

	return &inSubqueryCondition{
		column:   column,
		subquery: subquery,
		negate:   true,
	}
}

func (c *inSubqueryCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	subSQL, subArgs, err := c.subquery.Build()
	if err != nil {
		// This should not happen if validation was done in InSubquery/NotInSubquery.
		// However, the subquery might have been modified after creation.
		// Return an error SQL fragment instead of panicking.
		errMsg := fmt.Sprintf("subquery build failed: %v", err)
		return fmt.Sprintf("/* ERROR: %s */ 1=0", errMsg), nil, paramStart
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
// If the subquery fails to build, it returns an errorCondition instead of panicking.
func Exists(subquery *SelectBuilder) Condition {
	if _, _, err := subquery.Build(); err != nil {
		return &errorCondition{
			err: fmt.Errorf("invalid subquery for EXISTS: %w", err),
		}
	}

	return &existsCondition{subquery: subquery, negate: false}
}

// NotExists creates a condition for NOT EXISTS (SELECT ...).
// If the subquery fails to build, it returns an errorCondition instead of panicking.
func NotExists(subquery *SelectBuilder) Condition {
	if _, _, err := subquery.Build(); err != nil {
		return &errorCondition{
			err: fmt.Errorf("invalid subquery for NOT EXISTS: %w", err),
		}
	}

	return &existsCondition{subquery: subquery, negate: true}
}

func (c *existsCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	subSQL, subArgs, err := c.subquery.Build()
	if err != nil {
		// This should not happen if validation was done in Exists/NotExists.
		// However, the subquery might have been modified after creation.
		// Return an error SQL fragment instead of panicking.
		errMsg := fmt.Sprintf("subquery build failed: %v", err)
		return fmt.Sprintf("/* ERROR: %s */ 1=0", errMsg), nil, paramStart
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
