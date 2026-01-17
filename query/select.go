package query

import (
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

type SelectBuilder struct {
	dialect          database.Dialect
	columns          []string
	columnExprs      []Expression
	table            string
	tableAlias       string
	joins            []joinClause
	conditions       []Condition
	groupBy          []string
	groupByExprs     []Expression
	havingConditions []Condition
	orderBy          []orderClause
	orderByExprs     []orderByExpr
	limit            int
	offset           int
	distinct         bool
}

// orderByExpr represents an ORDER BY clause using an expression.
type orderByExpr struct {
	expr      Expression
	direction Order
}

func (s *SelectBuilder) From(table string) *SelectBuilder {
	s.table = table
	return s
}

func (s *SelectBuilder) As(alias string) *SelectBuilder {
	s.tableAlias = alias
	return s
}

func (s *SelectBuilder) Distinct() *SelectBuilder {
	s.distinct = true
	return s
}

func (s *SelectBuilder) Where(condition Condition) *SelectBuilder {
	s.conditions = append(s.conditions, condition)
	return s
}

func (s *SelectBuilder) And(condition Condition) *SelectBuilder {
	s.conditions = append(s.conditions, condition)
	return s
}

func (s *SelectBuilder) Or(condition Condition) *SelectBuilder {
	if len(s.conditions) == 0 {
		s.conditions = append(s.conditions, condition)
		return s
	}

	lastCondition := s.conditions[len(s.conditions)-1]
	s.conditions[len(s.conditions)-1] = Or(lastCondition, condition)
	return s
}

// Join adds an INNER JOIN clause to the query.
func (s *SelectBuilder) Join(table string, on Condition) *SelectBuilder {
	return s.InnerJoin(table, on)
}

// InnerJoin adds an INNER JOIN clause to the query.
func (s *SelectBuilder) InnerJoin(table string, on Condition) *SelectBuilder {
	s.joins = append(s.joins, joinClause{
		joinType:  InnerJoin,
		table:     table,
		condition: on,
	})
	return s
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (s *SelectBuilder) LeftJoin(table string, on Condition) *SelectBuilder {
	s.joins = append(s.joins, joinClause{
		joinType:  LeftJoin,
		table:     table,
		condition: on,
	})
	return s
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (s *SelectBuilder) RightJoin(table string, on Condition) *SelectBuilder {
	s.joins = append(s.joins, joinClause{
		joinType:  RightJoin,
		table:     table,
		condition: on,
	})
	return s
}

// FullJoin adds a FULL OUTER JOIN clause to the query.
// This will panic if the dialect doesn't support FULL JOIN (MySQL, SQLite).
func (s *SelectBuilder) FullJoin(table string, on Condition) *SelectBuilder {
	if !s.dialect.SupportsFullJoin() {
		panic("FULL JOIN not supported by this database dialect")
	}
	s.joins = append(s.joins, joinClause{
		joinType:  FullJoin,
		table:     table,
		condition: on,
	})
	return s
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (s *SelectBuilder) CrossJoin(table string) *SelectBuilder {
	s.joins = append(s.joins, joinClause{
		joinType: CrossJoin,
		table:    table,
	})
	return s
}

// JoinAs adds a JOIN clause with a table alias.
func (s *SelectBuilder) JoinAs(table, alias string, on Condition) *SelectBuilder {
	s.joins = append(s.joins, joinClause{
		joinType:  InnerJoin,
		table:     table,
		alias:     alias,
		condition: on,
	})
	return s
}

// LeftJoinAs adds a LEFT JOIN clause with a table alias.
func (s *SelectBuilder) LeftJoinAs(table, alias string, on Condition) *SelectBuilder {
	s.joins = append(s.joins, joinClause{
		joinType:  LeftJoin,
		table:     table,
		alias:     alias,
		condition: on,
	})
	return s
}

// SelectExpr adds expressions to the SELECT clause.
func (s *SelectBuilder) SelectExpr(exprs ...Expression) *SelectBuilder {
	s.columnExprs = append(s.columnExprs, exprs...)
	return s
}

func (s *SelectBuilder) OrderBy(column string, direction Order) *SelectBuilder {
	s.orderBy = append(s.orderBy, orderClause{
		column:    column,
		direction: direction,
	})
	return s
}

// OrderByExpr adds an expression-based ORDER BY clause.
func (s *SelectBuilder) OrderByExpr(expr Expression, direction Order) *SelectBuilder {
	s.orderByExprs = append(s.orderByExprs, orderByExpr{
		expr:      expr,
		direction: direction,
	})
	return s
}

// GroupBy adds column names to the GROUP BY clause.
func (s *SelectBuilder) GroupBy(columns ...string) *SelectBuilder {
	s.groupBy = append(s.groupBy, columns...)
	return s
}

// GroupByExpr adds expressions to the GROUP BY clause.
func (s *SelectBuilder) GroupByExpr(exprs ...Expression) *SelectBuilder {
	s.groupByExprs = append(s.groupByExprs, exprs...)
	return s
}

// Having adds a condition to the HAVING clause (for filtering aggregates).
func (s *SelectBuilder) Having(condition Condition) *SelectBuilder {
	s.havingConditions = append(s.havingConditions, condition)
	return s
}

func (s *SelectBuilder) Limit(limit int) *SelectBuilder {
	s.limit = limit
	return s
}

func (s *SelectBuilder) Offset(offset int) *SelectBuilder {
	s.offset = offset
	return s
}

func (s *SelectBuilder) Build() (query string, args []any) {
	var parts []string
	var allArgs []any
	paramCount := 1

	selectClause := "SELECT"
	if s.distinct {
		selectClause = "SELECT DISTINCT"
	}

	if len(s.columns) == 0 && len(s.columnExprs) == 0 {
		selectClause += " *"
	} else {
		var columnParts []string

		for _, col := range s.columns {
			columnParts = append(columnParts, s.dialect.QuoteIdentifier(col))
		}

		for _, expr := range s.columnExprs {
			exprSQL, exprArgs, nextParam := expr.ToSQL(s.dialect, paramCount)
			columnParts = append(columnParts, exprSQL)
			allArgs = append(allArgs, exprArgs...)
			paramCount = nextParam
		}

		selectClause += " " + strings.Join(columnParts, ", ")
	}
	parts = append(parts, selectClause)

	if s.table != "" {
		fromClause := "FROM " + s.dialect.QuoteIdentifier(s.table)
		if s.tableAlias != "" {
			fromClause += " AS " + s.dialect.QuoteIdentifier(s.tableAlias)
		}
		parts = append(parts, fromClause)
	}

	if len(s.joins) > 0 {
		for _, join := range s.joins {
			joinSQL, joinArgs, nextParam := join.toSQL(s.dialect, paramCount)
			parts = append(parts, joinSQL)
			allArgs = append(allArgs, joinArgs...)
			paramCount = nextParam
		}
	}

	if len(s.conditions) > 0 {
		var whereParts []string
		for _, cond := range s.conditions {
			sql, args, nextParam := cond.ToSQL(s.dialect, paramCount)
			whereParts = append(whereParts, sql)
			allArgs = append(allArgs, args...)
			paramCount = nextParam
		}
		whereClause := "WHERE " + strings.Join(whereParts, " AND ")
		parts = append(parts, whereClause)
	}

	if len(s.groupBy) > 0 || len(s.groupByExprs) > 0 {
		var groupParts []string

		for _, col := range s.groupBy {
			groupParts = append(groupParts, s.dialect.QuoteIdentifier(col))
		}

		for _, expr := range s.groupByExprs {
			exprSQL, exprArgs, nextParam := expr.ToSQL(s.dialect, paramCount)
			groupParts = append(groupParts, exprSQL)
			allArgs = append(allArgs, exprArgs...)
			paramCount = nextParam
		}

		groupClause := "GROUP BY " + strings.Join(groupParts, ", ")
		parts = append(parts, groupClause)
	}

	if len(s.havingConditions) > 0 {
		var havingParts []string
		for _, cond := range s.havingConditions {
			sql, args, nextParam := cond.ToSQL(s.dialect, paramCount)
			havingParts = append(havingParts, sql)
			allArgs = append(allArgs, args...)
			paramCount = nextParam
		}
		havingClause := "HAVING " + strings.Join(havingParts, " AND ")
		parts = append(parts, havingClause)
	}

	if len(s.orderBy) > 0 || len(s.orderByExprs) > 0 {
		var orderParts []string

		for _, order := range s.orderBy {
			orderParts = append(orderParts, s.dialect.QuoteIdentifier(order.column)+" "+order.direction.String())
		}

		for _, order := range s.orderByExprs {
			exprSQL, exprArgs, nextParam := order.expr.ToSQL(s.dialect, paramCount)
			orderParts = append(orderParts, exprSQL+" "+order.direction.String())
			allArgs = append(allArgs, exprArgs...)
			paramCount = nextParam
		}

		orderClause := "ORDER BY " + strings.Join(orderParts, ", ")
		parts = append(parts, orderClause)
	}

	if s.limit > 0 || s.offset > 0 {
		limitOffsetClause := s.dialect.LimitOffset(s.limit, s.offset)
		if limitOffsetClause != "" {
			parts = append(parts, limitOffsetClause)
		}
	}

	query = strings.Join(parts, " ")
	return query, allArgs
}
