package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

type Condition interface {
	ToSQL(dialect database.Dialect, paramStart int) (sql string, args []any, nextParam int)
}

type comparisonCondition struct {
	column   string
	operator string
	value    any
}

func (c *comparisonCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	sql := fmt.Sprintf("%s %s %s", dialect.QuoteIdentifier(c.column), c.operator, dialect.Placeholder(paramStart))
	return sql, []any{c.value}, paramStart + 1
}

func Eq(column string, value any) Condition {
	return &comparisonCondition{column: column, operator: "=", value: value}
}

func Ne(column string, value any) Condition {
	return &comparisonCondition{column: column, operator: "!=", value: value}
}

func Gt(column string, value any) Condition {
	return &comparisonCondition{column: column, operator: ">", value: value}
}

func Gte(column string, value any) Condition {
	return &comparisonCondition{column: column, operator: ">=", value: value}
}

func Lt(column string, value any) Condition {
	return &comparisonCondition{column: column, operator: "<", value: value}
}

func Lte(column string, value any) Condition {
	return &comparisonCondition{column: column, operator: "<=", value: value}
}

type likeCondition struct {
	column  string
	pattern string
	notLike bool
}

func (c *likeCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	operator := "LIKE"
	if c.notLike {
		operator = "NOT LIKE"
	}
	sql := fmt.Sprintf("%s %s %s", dialect.QuoteIdentifier(c.column), operator, dialect.Placeholder(paramStart))
	return sql, []any{c.pattern}, paramStart + 1
}

func Like(column string, pattern string) Condition {
	return &likeCondition{column: column, pattern: pattern, notLike: false}
}

func NotLike(column string, pattern string) Condition {
	return &likeCondition{column: column, pattern: pattern, notLike: true}
}

type ilikeCondition struct {
	column  string
	pattern string
}

func (c *ilikeCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	caseInsensitiveOp := dialect.CaseInsensitiveLike()

	if caseInsensitiveOp == "ILIKE" {
		sql := fmt.Sprintf("%s ILIKE %s", dialect.QuoteIdentifier(c.column), dialect.Placeholder(paramStart))
		return sql, []any{c.pattern}, paramStart + 1
	}

	sql := fmt.Sprintf("LOWER(%s) LIKE LOWER(%s)", dialect.QuoteIdentifier(c.column), dialect.Placeholder(paramStart))
	return sql, []any{c.pattern}, paramStart + 1
}

func ILike(column string, pattern string) Condition {
	return &ilikeCondition{column: column, pattern: pattern}
}

type nullCondition struct {
	column string
	isNull bool
}

func (c *nullCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	operator := "IS NULL"
	if !c.isNull {
		operator = "IS NOT NULL"
	}
	sql := fmt.Sprintf("%s %s", dialect.QuoteIdentifier(c.column), operator)
	return sql, nil, paramStart
}

func IsNull(column string) Condition {
	return &nullCondition{column: column, isNull: true}
}

func IsNotNull(column string) Condition {
	return &nullCondition{column: column, isNull: false}
}

type inCondition struct {
	column string
	values []any
	notIn  bool
}

func (c *inCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	if len(c.values) == 0 {
		if c.notIn {
			return "1=1", nil, paramStart
		}
		return "1=0", nil, paramStart
	}

	placeholders := make([]string, len(c.values))
	for i := range c.values {
		placeholders[i] = dialect.Placeholder(paramStart + i)
	}

	operator := "IN"
	if c.notIn {
		operator = "NOT IN"
	}

	sql := fmt.Sprintf("%s %s (%s)", dialect.QuoteIdentifier(c.column), operator, strings.Join(placeholders, ", "))
	return sql, c.values, paramStart + len(c.values)
}

func In(column string, values ...any) Condition {
	return &inCondition{column: column, values: values, notIn: false}
}

func NotIn(column string, values ...any) Condition {
	return &inCondition{column: column, values: values, notIn: true}
}

type betweenCondition struct {
	column string
	start  any
	end    any
}

func (c *betweenCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	sql := fmt.Sprintf("%s BETWEEN %s AND %s",
		dialect.QuoteIdentifier(c.column),
		dialect.Placeholder(paramStart),
		dialect.Placeholder(paramStart+1))
	return sql, []any{c.start, c.end}, paramStart + 2
}

func Between(column string, start, end any) Condition {
	return &betweenCondition{column: column, start: start, end: end}
}

type logicalCondition struct {
	operator   string
	conditions []Condition
}

func (c *logicalCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	if len(c.conditions) == 0 {
		return "1=1", nil, paramStart
	}

	if len(c.conditions) == 1 {
		return c.conditions[0].ToSQL(dialect, paramStart)
	}

	var parts []string
	var allArgs []any
	currentParam := paramStart

	for _, cond := range c.conditions {
		sql, args, nextParam := cond.ToSQL(dialect, currentParam)
		parts = append(parts, sql)
		allArgs = append(allArgs, args...)
		currentParam = nextParam
	}

	sql := "(" + strings.Join(parts, " "+c.operator+" ") + ")"
	return sql, allArgs, currentParam
}

func And(conditions ...Condition) Condition {
	return &logicalCondition{operator: "AND", conditions: conditions}
}

func Or(conditions ...Condition) Condition {
	return &logicalCondition{operator: "OR", conditions: conditions}
}

type notCondition struct {
	condition Condition
}

func (c *notCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	sql, args, nextParam := c.condition.ToSQL(dialect, paramStart)
	return "NOT (" + sql + ")", args, nextParam
}

func Not(condition Condition) Condition {
	return &notCondition{condition: condition}
}

type rawCondition struct {
	sql  string
	args []any
}

func (c *rawCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	sql := c.sql
	currentParam := paramStart

	for i := 0; i < len(c.args); i++ {
		sql = strings.Replace(sql, "?", dialect.Placeholder(currentParam), 1)
		currentParam++
	}

	return sql, c.args, currentParam
}

func Raw(sql string, args ...any) Condition {
	return &rawCondition{sql: sql, args: args}
}

type columnComparisonCondition struct {
	col1     string
	col2     string
	operator string
}

func (c *columnComparisonCondition) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	sql := fmt.Sprintf("%s %s %s", dialect.QuoteIdentifier(c.col1), c.operator, dialect.QuoteIdentifier(c.col2))
	return sql, nil, paramStart
}

func ColEq(col1, col2 string) Condition {
	return &columnComparisonCondition{col1: col1, col2: col2, operator: "="}
}

func ColNe(col1, col2 string) Condition {
	return &columnComparisonCondition{col1: col1, col2: col2, operator: "!="}
}

func ColGt(col1, col2 string) Condition {
	return &columnComparisonCondition{col1: col1, col2: col2, operator: ">"}
}

func ColLt(col1, col2 string) Condition {
	return &columnComparisonCondition{col1: col1, col2: col2, operator: "<"}
}
