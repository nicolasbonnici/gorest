package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

// WindowBuilder builds a window specification (OVER clause).
type WindowBuilder struct {
	partitionBy       []Expression
	orderBy           []orderByExpr
	frameType         string // "ROWS" or "RANGE"
	frameStart        string
	frameEnd          string
	excludeCurrentRow bool
}

// Window creates a new window specification builder.
func Window() *WindowBuilder {
	return &WindowBuilder{
		partitionBy: make([]Expression, 0),
		orderBy:     make([]orderByExpr, 0),
	}
}

// PartitionBy adds PARTITION BY expressions to the window.
func (w *WindowBuilder) PartitionBy(exprs ...Expression) *WindowBuilder {
	w.partitionBy = append(w.partitionBy, exprs...)
	return w
}

// OrderBy adds an ORDER BY clause to the window.
func (w *WindowBuilder) OrderBy(expr Expression, direction Order) *WindowBuilder {
	w.orderBy = append(w.orderBy, orderByExpr{
		expr:      expr,
		direction: direction,
	})
	return w
}

// RowsFrame specifies a ROWS frame for the window.
func (w *WindowBuilder) RowsFrame(start, end string) *WindowBuilder {
	w.frameType = "ROWS"
	w.frameStart = start
	w.frameEnd = end
	return w
}

// RangeFrame specifies a RANGE frame for the window.
func (w *WindowBuilder) RangeFrame(start, end string) *WindowBuilder {
	w.frameType = "RANGE"
	w.frameStart = start
	w.frameEnd = end
	return w
}

// RowsBetween is a helper for common frame specifications.
func (w *WindowBuilder) RowsBetween(start, end string) *WindowBuilder {
	return w.RowsFrame(start, end)
}

// toSQL converts the window specification to SQL.
func (w *WindowBuilder) toSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var parts []string
	var allArgs []any
	currentParam := paramStart

	if len(w.partitionBy) > 0 {
		var partitionParts []string
		for _, expr := range w.partitionBy {
			exprSQL, exprArgs, nextParam := expr.ToSQL(dialect, currentParam)
			partitionParts = append(partitionParts, exprSQL)
			allArgs = append(allArgs, exprArgs...)
			currentParam = nextParam
		}
		parts = append(parts, "PARTITION BY "+strings.Join(partitionParts, ", "))
	}

	if len(w.orderBy) > 0 {
		var orderParts []string
		for _, order := range w.orderBy {
			exprSQL, exprArgs, nextParam := order.expr.ToSQL(dialect, currentParam)
			orderParts = append(orderParts, exprSQL+" "+order.direction.String())
			allArgs = append(allArgs, exprArgs...)
			currentParam = nextParam
		}
		parts = append(parts, "ORDER BY "+strings.Join(orderParts, ", "))
	}

	if w.frameType != "" {
		frameSpec := fmt.Sprintf("%s BETWEEN %s AND %s", w.frameType, w.frameStart, w.frameEnd)
		parts = append(parts, frameSpec)
	}

	if len(parts) == 0 {
		return "", nil, currentParam
	}

	return strings.Join(parts, " "), allArgs, currentParam
}

// Window Function Expressions

// windowFuncExpr represents a window function expression.
type windowFuncExpr struct {
	funcName string
	args     []Expression
	window   *WindowBuilder
}

func (w *windowFuncExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var allArgs []any
	currentParam := paramStart

	var argParts []string
	for _, arg := range w.args {
		argSQL, argArgs, nextParam := arg.ToSQL(dialect, currentParam)
		argParts = append(argParts, argSQL)
		allArgs = append(allArgs, argArgs...)
		currentParam = nextParam
	}

	windowSQL, windowArgs, nextParam := w.window.toSQL(dialect, currentParam)
	allArgs = append(allArgs, windowArgs...)
	currentParam = nextParam

	funcCall := fmt.Sprintf("%s(%s)", w.funcName, strings.Join(argParts, ", "))
	if windowSQL != "" {
		return fmt.Sprintf("%s OVER (%s)", funcCall, windowSQL), allArgs, currentParam
	}
	return fmt.Sprintf("%s OVER ()", funcCall), allArgs, currentParam
}

// Ranking Functions

// RowNumber creates a ROW_NUMBER() window function.
func RowNumber(window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "ROW_NUMBER",
		args:     []Expression{},
		window:   window,
	}
}

// Rank creates a RANK() window function.
func Rank(window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "RANK",
		args:     []Expression{},
		window:   window,
	}
}

// DenseRank creates a DENSE_RANK() window function.
func DenseRank(window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "DENSE_RANK",
		args:     []Expression{},
		window:   window,
	}
}

// Ntile creates an NTILE(n) window function.
func Ntile(n Expression, window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "NTILE",
		args:     []Expression{n},
		window:   window,
	}
}

// Value Functions

// FirstValue creates a FIRST_VALUE() window function.
func FirstValue(expr Expression, window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "FIRST_VALUE",
		args:     []Expression{expr},
		window:   window,
	}
}

// LastValue creates a LAST_VALUE() window function.
func LastValue(expr Expression, window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "LAST_VALUE",
		args:     []Expression{expr},
		window:   window,
	}
}

// NthValue creates an NTH_VALUE() window function.
func NthValue(expr, n Expression, window *WindowBuilder) Expression {
	return &windowFuncExpr{
		funcName: "NTH_VALUE",
		args:     []Expression{expr, n},
		window:   window,
	}
}

// Offset Functions

// Lag creates a LAG() window function.
// offset and default are optional (can be nil).
func Lag(expr, offset, defaultValue Expression, window *WindowBuilder) Expression {
	args := []Expression{expr}
	if offset != nil {
		args = append(args, offset)
		if defaultValue != nil {
			args = append(args, defaultValue)
		}
	}
	return &windowFuncExpr{
		funcName: "LAG",
		args:     args,
		window:   window,
	}
}

// Lead creates a LEAD() window function.
// offset and default are optional (can be nil).
func Lead(expr, offset, defaultValue Expression, window *WindowBuilder) Expression {
	args := []Expression{expr}
	if offset != nil {
		args = append(args, offset)
		if defaultValue != nil {
			args = append(args, defaultValue)
		}
	}
	return &windowFuncExpr{
		funcName: "LEAD",
		args:     args,
		window:   window,
	}
}

// Aggregate Window Functions
// These use the existing aggregate functions but with a window specification.

// windowAggregateExpr wraps an aggregate function with a window.
type windowAggregateExpr struct {
	aggExpr Expression
	window  *WindowBuilder
}

func (w *windowAggregateExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var allArgs []any
	currentParam := paramStart

	aggSQL, aggArgs, nextParam := w.aggExpr.ToSQL(dialect, currentParam)
	allArgs = append(allArgs, aggArgs...)
	currentParam = nextParam

	windowSQL, windowArgs, nextParam := w.window.toSQL(dialect, currentParam)
	allArgs = append(allArgs, windowArgs...)
	currentParam = nextParam

	if windowSQL != "" {
		return fmt.Sprintf("%s OVER (%s)", aggSQL, windowSQL), allArgs, currentParam
	}
	return fmt.Sprintf("%s OVER ()", aggSQL), allArgs, currentParam
}

// SumOver creates a SUM() window function.
func SumOver(expr Expression, window *WindowBuilder) Expression {
	return &windowAggregateExpr{
		aggExpr: Sum(expr),
		window:  window,
	}
}

// AvgOver creates an AVG() window function.
func AvgOver(expr Expression, window *WindowBuilder) Expression {
	return &windowAggregateExpr{
		aggExpr: Avg(expr),
		window:  window,
	}
}

// CountOver creates a COUNT() window function.
func CountOver(expr Expression, window *WindowBuilder) Expression {
	return &windowAggregateExpr{
		aggExpr: Count(expr),
		window:  window,
	}
}

// MinOver creates a MIN() window function.
func MinOver(expr Expression, window *WindowBuilder) Expression {
	return &windowAggregateExpr{
		aggExpr: Min(expr),
		window:  window,
	}
}

// MaxOver creates a MAX() window function.
func MaxOver(expr Expression, window *WindowBuilder) Expression {
	return &windowAggregateExpr{
		aggExpr: Max(expr),
		window:  window,
	}
}

// Frame boundary constants for convenience.
const (
	UnboundedPreceding = "UNBOUNDED PRECEDING"
	CurrentRow         = "CURRENT ROW"
	UnboundedFollowing = "UNBOUNDED FOLLOWING"
)
