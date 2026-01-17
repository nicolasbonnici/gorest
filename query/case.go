package query

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

type CaseBuilder struct {
	caseExpr Expression
	whens    []whenClause
	elseExpr Expression
}

type whenClause struct {
	condition Condition
	value     Expression
	result    Expression
}

func Case(expr Expression) *CaseBuilder {
	return &CaseBuilder{
		caseExpr: expr,
		whens:    make([]whenClause, 0),
	}
}

func (c *CaseBuilder) When(value Expression, result Expression) *CaseBuilder {
	c.whens = append(c.whens, whenClause{
		value:  value,
		result: result,
	})
	return c
}

func (c *CaseBuilder) WhenCond(condition Condition, result Expression) *CaseBuilder {
	c.whens = append(c.whens, whenClause{
		condition: condition,
		result:    result,
	})
	return c
}

func (c *CaseBuilder) Else(expr Expression) *CaseBuilder {
	c.elseExpr = expr
	return c
}

func (c *CaseBuilder) End() Expression {
	return &caseExpr{builder: c}
}

type caseExpr struct {
	builder *CaseBuilder
}

func (c *caseExpr) ToSQL(dialect database.Dialect, paramStart int) (string, []any, int) {
	var parts []string
	var allArgs []any
	currentParam := paramStart

	if c.builder.caseExpr != nil {
		caseSQL, caseArgs, nextParam := c.builder.caseExpr.ToSQL(dialect, currentParam)
		parts = append(parts, "CASE "+caseSQL)
		allArgs = append(allArgs, caseArgs...)
		currentParam = nextParam

		for _, when := range c.builder.whens {
			valueSQL, valueArgs, param1 := when.value.ToSQL(dialect, currentParam)
			resultSQL, resultArgs, param2 := when.result.ToSQL(dialect, param1)

			parts = append(parts, fmt.Sprintf("WHEN %s THEN %s", valueSQL, resultSQL))
			allArgs = append(allArgs, valueArgs...)
			allArgs = append(allArgs, resultArgs...)
			currentParam = param2
		}
	} else {
		parts = append(parts, "CASE")

		for _, when := range c.builder.whens {
			condSQL, condArgs, param1 := when.condition.ToSQL(dialect, currentParam)
			resultSQL, resultArgs, param2 := when.result.ToSQL(dialect, param1)

			parts = append(parts, fmt.Sprintf("WHEN %s THEN %s", condSQL, resultSQL))
			allArgs = append(allArgs, condArgs...)
			allArgs = append(allArgs, resultArgs...)
			currentParam = param2
		}
	}

	if c.builder.elseExpr != nil {
		elseSQL, elseArgs, nextParam := c.builder.elseExpr.ToSQL(dialect, currentParam)
		parts = append(parts, "ELSE "+elseSQL)
		allArgs = append(allArgs, elseArgs...)
		currentParam = nextParam
	}

	parts = append(parts, "END")
	return strings.Join(parts, " "), allArgs, currentParam
}
