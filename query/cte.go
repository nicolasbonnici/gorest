package query

import (
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

type CTE struct {
	name      string
	columns   []string
	query     *SelectBuilder
	recursive bool
}

type CTEBuilder struct {
	ctes    []*CTE
	builder *SelectBuilder
}

func (b *Builder) WithCTE(name string, query *SelectBuilder, columns ...string) *CTEBuilder {
	return &CTEBuilder{
		ctes: []*CTE{
			{
				name:      name,
				columns:   columns,
				query:     query,
				recursive: false,
			},
		},
		builder: &SelectBuilder{
			dialect:          b.dialect,
			columns:          make([]string, 0),
			columnExprs:      make([]Expression, 0),
			joins:            make([]joinClause, 0),
			conditions:       make([]Condition, 0),
			groupBy:          make([]string, 0),
			groupByExprs:     make([]Expression, 0),
			havingConditions: make([]Condition, 0),
			orderBy:          make([]orderClause, 0),
			orderByExprs:     make([]orderByExpr, 0),
		},
	}
}

func (b *Builder) WithRecursiveCTE(name string, query *SelectBuilder, columns ...string) *CTEBuilder {
	return &CTEBuilder{
		ctes: []*CTE{
			{
				name:      name,
				columns:   columns,
				query:     query,
				recursive: true,
			},
		},
		builder: &SelectBuilder{
			dialect:          b.dialect,
			columns:          make([]string, 0),
			columnExprs:      make([]Expression, 0),
			joins:            make([]joinClause, 0),
			conditions:       make([]Condition, 0),
			groupBy:          make([]string, 0),
			groupByExprs:     make([]Expression, 0),
			havingConditions: make([]Condition, 0),
			orderBy:          make([]orderClause, 0),
			orderByExprs:     make([]orderByExpr, 0),
		},
	}
}

func (c *CTEBuilder) AndCTE(name string, query *SelectBuilder, columns ...string) *CTEBuilder {
	c.ctes = append(c.ctes, &CTE{
		name:      name,
		columns:   columns,
		query:     query,
		recursive: false,
	})
	return c
}

func (c *CTEBuilder) Select(columns ...string) *CTEBuilder {
	c.builder.columns = columns
	return c
}

func (c *CTEBuilder) SelectExpr(exprs ...Expression) *CTEBuilder {
	c.builder.columnExprs = append(c.builder.columnExprs, exprs...)
	return c
}

func (c *CTEBuilder) From(table string) *CTEBuilder {
	c.builder.table = table
	return c
}

func (c *CTEBuilder) As(alias string) *CTEBuilder {
	c.builder.tableAlias = alias
	return c
}

func (c *CTEBuilder) Join(table string, on Condition) *CTEBuilder {
	c.builder = c.builder.Join(table, on)
	return c
}

func (c *CTEBuilder) JoinAs(table, alias string, on Condition) *CTEBuilder {
	c.builder = c.builder.JoinAs(table, alias, on)
	return c
}

func (c *CTEBuilder) LeftJoin(table string, on Condition) *CTEBuilder {
	c.builder = c.builder.LeftJoin(table, on)
	return c
}

func (c *CTEBuilder) LeftJoinAs(table, alias string, on Condition) *CTEBuilder {
	c.builder = c.builder.LeftJoinAs(table, alias, on)
	return c
}

func (c *CTEBuilder) Where(condition Condition) *CTEBuilder {
	c.builder = c.builder.Where(condition)
	return c
}

func (c *CTEBuilder) GroupBy(columns ...string) *CTEBuilder {
	c.builder = c.builder.GroupBy(columns...)
	return c
}

func (c *CTEBuilder) Having(condition Condition) *CTEBuilder {
	c.builder = c.builder.Having(condition)
	return c
}

func (c *CTEBuilder) OrderBy(column string, direction Order) *CTEBuilder {
	c.builder = c.builder.OrderBy(column, direction)
	return c
}

func (c *CTEBuilder) Limit(limit int) *CTEBuilder {
	c.builder = c.builder.Limit(limit)
	return c
}

func (c *CTEBuilder) Offset(offset int) *CTEBuilder {
	c.builder = c.builder.Offset(offset)
	return c
}

func (c *CTEBuilder) Build() (query string, args []any, err error) {
	var parts []string
	var allArgs []any
	currentParam := 1

	var cteParts []string
	hasRecursive := false

	for _, cte := range c.ctes {
		if cte.recursive {
			hasRecursive = true
		}

		cteSQL, cteArgs, cteErr := cte.query.Build()
		if cteErr != nil {
			return "", nil, cteErr
		}

		if len(cteArgs) > 0 {
			cteSQL = renumberQueryParameters(c.builder.dialect, cteSQL, len(cteArgs), currentParam)
			allArgs = append(allArgs, cteArgs...)
			currentParam += len(cteArgs)
		}

		var cteDef string
		if len(cte.columns) > 0 {
			quotedCols := make([]string, len(cte.columns))
			for i, col := range cte.columns {
				quotedCols[i] = c.builder.dialect.QuoteIdentifier(col)
			}
			cteDef = c.builder.dialect.QuoteIdentifier(cte.name) + " (" + strings.Join(quotedCols, ", ") + ") AS (" + cteSQL + ")"
		} else {
			cteDef = c.builder.dialect.QuoteIdentifier(cte.name) + " AS (" + cteSQL + ")"
		}

		cteParts = append(cteParts, cteDef)
	}

	withClause := "WITH "
	if hasRecursive {
		withClause = "WITH RECURSIVE "
	}
	withClause += strings.Join(cteParts, ", ")
	parts = append(parts, withClause)

	mainSQL, mainArgs, mainErr := c.builder.Build()
	if mainErr != nil {
		return "", nil, mainErr
	}

	if len(mainArgs) > 0 {
		mainSQL = renumberQueryParameters(c.builder.dialect, mainSQL, len(mainArgs), currentParam)
		allArgs = append(allArgs, mainArgs...)
	}

	parts = append(parts, mainSQL)

	return strings.Join(parts, " "), allArgs, nil
}

func renumberQueryParameters(dialect database.Dialect, sql string, argCount int, startFrom int) string {
	result := sql

	if dialect.Placeholder(1) == "?" {
		return result
	}

	// Replace from highest to lowest to avoid conflicts
	for i := argCount; i >= 1; i-- {
		oldPlaceholder := dialect.Placeholder(i)
		newPlaceholder := dialect.Placeholder(startFrom + i - 1)
		result = strings.Replace(result, oldPlaceholder, newPlaceholder, 1)
	}

	return result
}
