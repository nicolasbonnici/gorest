package query

import (
	"fmt"

	"github.com/nicolasbonnici/gorest/database"
)

type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullJoin
	CrossJoin
)

type joinClause struct {
	joinType  JoinType
	table     string
	alias     string
	condition Condition
}

func (j *joinClause) toSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var joinType string
	switch j.joinType {
	case InnerJoin:
		joinType = "INNER JOIN"
	case LeftJoin:
		joinType = "LEFT JOIN"
	case RightJoin:
		joinType = "RIGHT JOIN"
	case FullJoin:
		joinType = "FULL OUTER JOIN"
	case CrossJoin:
		joinType = "CROSS JOIN"
	}

	tableExpr := dialect.QuoteIdentifier(j.table)
	if j.alias != "" {
		tableExpr += " AS " + dialect.QuoteIdentifier(j.alias)
	}

	if j.joinType == CrossJoin {
		return fmt.Sprintf("%s %s", joinType, tableExpr), nil, paramStart
	}

	onSQL, onArgs, nextParam := j.condition.ToSQL(dialect, paramStart)
	return fmt.Sprintf("%s %s ON %s", joinType, tableExpr, onSQL), onArgs, nextParam
}
