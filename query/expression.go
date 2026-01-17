package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

// Expression represents a SQL expression (function, column reference, literal, etc.)
// that can be used in SELECT, WHERE, HAVING, and ORDER BY clauses.
type Expression interface {
	// ToSQL converts the expression to SQL with proper parameter handling.
	ToSQL(dialect database.Dialect, paramStart int) (sql string, args []any, nextParam int)
}

// rawExpr represents a raw SQL expression with optional parameters.
type rawExpr struct {
	sql  string
	args []any
}

// RawExpr creates a raw SQL expression.
func RawExpr(sql string, args ...any) Expression {
	return &rawExpr{sql: sql, args: args}
}

func (r *rawExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	result := r.sql
	for i := 0; i < len(r.args); i++ {
		placeholder := dialect.Placeholder(paramStart + i)
		result = strings.Replace(result, "?", placeholder, 1)
	}
	return result, r.args, paramStart + len(r.args)
}

// columnExpr represents a simple column reference.
type columnExpr struct {
	column string
}

// Col creates a column reference expression.
func Col(column string) Expression {
	return &columnExpr{column: column}
}

func (c *columnExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	return dialect.QuoteIdentifier(c.column), nil, paramStart
}

// literalExpr represents a literal value (parameterized).
type literalExpr struct {
	value any
}

// Literal creates a literal value expression (parameterized).
func Literal(value any) Expression {
	return &literalExpr{value: value}
}

func (l *literalExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	return dialect.Placeholder(paramStart), []any{l.value}, paramStart + 1
}

// Aggregate Functions

// countExpr represents COUNT function.
type countExpr struct {
	expr     Expression
	distinct bool
}

// Count creates a COUNT expression.
// Use Count(Col("*")) for COUNT(*), Count(Col("id")) for COUNT(id), Count(Distinct(Col("id"))) for COUNT(DISTINCT id).
func Count(expr Expression) Expression {
	return &countExpr{expr: expr, distinct: false}
}

// CountDistinct creates a COUNT(DISTINCT ...) expression.
func CountDistinct(expr Expression) Expression {
	return &countExpr{expr: expr, distinct: true}
}

func (c *countExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := c.expr.ToSQL(dialect, paramStart)

	if c.distinct {
		return fmt.Sprintf("COUNT(DISTINCT %s)", exprSQL), args, nextParam
	}
	return fmt.Sprintf("COUNT(%s)", exprSQL), args, nextParam
}

// sumExpr represents SUM function.
type sumExpr struct {
	expr Expression
}

// Sum creates a SUM expression.
func Sum(expr Expression) Expression {
	return &sumExpr{expr: expr}
}

func (s *sumExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := s.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("SUM(%s)", exprSQL), args, nextParam
}

// avgExpr represents AVG function.
type avgExpr struct {
	expr Expression
}

// Avg creates an AVG expression.
func Avg(expr Expression) Expression {
	return &avgExpr{expr: expr}
}

func (a *avgExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := a.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("AVG(%s)", exprSQL), args, nextParam
}

// minExpr represents MIN function.
type minExpr struct {
	expr Expression
}

// Min creates a MIN expression.
func Min(expr Expression) Expression {
	return &minExpr{expr: expr}
}

func (m *minExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := m.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("MIN(%s)", exprSQL), args, nextParam
}

// maxExpr represents MAX function.
type maxExpr struct {
	expr Expression
}

// Max creates a MAX expression.
func Max(expr Expression) Expression {
	return &maxExpr{expr: expr}
}

func (m *maxExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := m.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("MAX(%s)", exprSQL), args, nextParam
}

// String Functions

// concatExpr represents CONCAT function.
type concatExpr struct {
	exprs []Expression
}

// Concat creates a CONCAT expression.
func Concat(exprs ...Expression) Expression {
	return &concatExpr{exprs: exprs}
}

func (c *concatExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var parts []string
	var allArgs []any
	currentParam := paramStart

	for _, expr := range c.exprs {
		sql, args, nextParam := expr.ToSQL(dialect, currentParam)
		parts = append(parts, sql)
		allArgs = append(allArgs, args...)
		currentParam = nextParam
	}

	return fmt.Sprintf("CONCAT(%s)", strings.Join(parts, ", ")), allArgs, currentParam
}

// upperExpr represents UPPER function.
type upperExpr struct {
	expr Expression
}

// Upper creates an UPPER expression.
func Upper(expr Expression) Expression {
	return &upperExpr{expr: expr}
}

func (u *upperExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := u.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("UPPER(%s)", exprSQL), args, nextParam
}

// lowerExpr represents LOWER function.
type lowerExpr struct {
	expr Expression
}

// Lower creates a LOWER expression.
func Lower(expr Expression) Expression {
	return &lowerExpr{expr: expr}
}

func (l *lowerExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := l.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("LOWER(%s)", exprSQL), args, nextParam
}

// lengthExpr represents LENGTH/LEN function.
type lengthExpr struct {
	expr Expression
}

// Length creates a LENGTH expression.
func Length(expr Expression) Expression {
	return &lengthExpr{expr: expr}
}

func (l *lengthExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := l.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("LENGTH(%s)", exprSQL), args, nextParam
}

// trimExpr represents TRIM function.
type trimExpr struct {
	expr Expression
}

// Trim creates a TRIM expression.
func Trim(expr Expression) Expression {
	return &trimExpr{expr: expr}
}

func (t *trimExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := t.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("TRIM(%s)", exprSQL), args, nextParam
}

// substringExpr represents SUBSTRING function.
type substringExpr struct {
	expr  Expression
	start Expression
	len   Expression
}

// Substring creates a SUBSTRING expression.
// If len is nil, extracts from start to the end.
func Substring(expr, start, length Expression) Expression {
	return &substringExpr{expr: expr, start: start, len: length}
}

func (s *substringExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args1, param1 := s.expr.ToSQL(dialect, paramStart)
	startSQL, args2, param2 := s.start.ToSQL(dialect, param1)

	var allArgs []any
	allArgs = append(allArgs, args1...)
	allArgs = append(allArgs, args2...)

	if s.len != nil {
		lenSQL, args3, param3 := s.len.ToSQL(dialect, param2)
		allArgs = append(allArgs, args3...)
		return fmt.Sprintf("SUBSTRING(%s, %s, %s)", exprSQL, startSQL, lenSQL), allArgs, param3
	}

	return fmt.Sprintf("SUBSTRING(%s, %s)", exprSQL, startSQL), allArgs, param2
}

// Math Functions

// absExpr represents ABS function.
type absExpr struct {
	expr Expression
}

// Abs creates an ABS expression.
func Abs(expr Expression) Expression {
	return &absExpr{expr: expr}
}

func (a *absExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := a.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("ABS(%s)", exprSQL), args, nextParam
}

// roundExpr represents ROUND function.
type roundExpr struct {
	expr      Expression
	precision Expression
}

// Round creates a ROUND expression.
// If precision is nil, rounds to integer.
func Round(expr, precision Expression) Expression {
	return &roundExpr{expr: expr, precision: precision}
}

func (r *roundExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args1, param1 := r.expr.ToSQL(dialect, paramStart)

	if r.precision != nil {
		precSQL, args2, param2 := r.precision.ToSQL(dialect, param1)
		allArgs := append(args1, args2...)
		return fmt.Sprintf("ROUND(%s, %s)", exprSQL, precSQL), allArgs, param2
	}

	return fmt.Sprintf("ROUND(%s)", exprSQL), args1, param1
}

// ceilExpr represents CEIL/CEILING function.
type ceilExpr struct {
	expr Expression
}

// Ceil creates a CEIL expression.
func Ceil(expr Expression) Expression {
	return &ceilExpr{expr: expr}
}

func (c *ceilExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := c.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("CEIL(%s)", exprSQL), args, nextParam
}

// floorExpr represents FLOOR function.
type floorExpr struct {
	expr Expression
}

// Floor creates a FLOOR expression.
func Floor(expr Expression) Expression {
	return &floorExpr{expr: expr}
}

func (f *floorExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := f.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("FLOOR(%s)", exprSQL), args, nextParam
}

// Date/Time Functions

// nowExpr represents NOW() function.
type nowExpr struct{}

// Now creates a NOW() expression.
func Now() Expression {
	return &nowExpr{}
}

func (n *nowExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	return "NOW()", nil, paramStart
}

// currentDateExpr represents CURRENT_DATE function.
type currentDateExpr struct{}

// CurrentDate creates a CURRENT_DATE expression.
func CurrentDate() Expression {
	return &currentDateExpr{}
}

func (c *currentDateExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	return "CURRENT_DATE", nil, paramStart
}

// currentTimeExpr represents CURRENT_TIME function.
type currentTimeExpr struct{}

// CurrentTime creates a CURRENT_TIME expression.
func CurrentTime() Expression {
	return &currentTimeExpr{}
}

func (c *currentTimeExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	return "CURRENT_TIME", nil, paramStart
}

// Conditional Functions

// coalesceExpr represents COALESCE function.
type coalesceExpr struct {
	exprs []Expression
}

// Coalesce creates a COALESCE expression.
func Coalesce(exprs ...Expression) Expression {
	return &coalesceExpr{exprs: exprs}
}

func (c *coalesceExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var parts []string
	var allArgs []any
	currentParam := paramStart

	for _, expr := range c.exprs {
		sql, args, nextParam := expr.ToSQL(dialect, currentParam)
		parts = append(parts, sql)
		allArgs = append(allArgs, args...)
		currentParam = nextParam
	}

	return fmt.Sprintf("COALESCE(%s)", strings.Join(parts, ", ")), allArgs, currentParam
}

// nullifExpr represents NULLIF function.
type nullifExpr struct {
	expr1 Expression
	expr2 Expression
}

// Nullif creates a NULLIF expression.
func Nullif(expr1, expr2 Expression) Expression {
	return &nullifExpr{expr1: expr1, expr2: expr2}
}

func (n *nullifExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	sql1, args1, param1 := n.expr1.ToSQL(dialect, paramStart)
	sql2, args2, param2 := n.expr2.ToSQL(dialect, param1)

	allArgs := append(args1, args2...)
	return fmt.Sprintf("NULLIF(%s, %s)", sql1, sql2), allArgs, param2
}

// aliasExpr represents an expression with an AS alias.
type aliasExpr struct {
	expr  Expression
	alias string
}

// As creates an aliased expression (for use in SELECT).
func (e *columnExpr) As(alias string) Expression {
	return &aliasExpr{expr: e, alias: alias}
}

// As adds an alias to any expression.
func As(expr Expression, alias string) Expression {
	return &aliasExpr{expr: expr, alias: alias}
}

func (a *aliasExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	exprSQL, args, nextParam := a.expr.ToSQL(dialect, paramStart)
	return fmt.Sprintf("%s AS %s", exprSQL, dialect.QuoteIdentifier(a.alias)), args, nextParam
}
