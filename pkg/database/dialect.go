package database

import "fmt"

type Dialect interface {
	Placeholder(n int) string
	SupportsReturning() bool
	ReturningClause(cols ...string) string
	LimitOffset(limit, offset int) string
	QuoteIdentifier(name string) string
	MapType(dbType string) string
}

type BaseDialect struct{}

func (d *BaseDialect) LimitOffset(limit, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	} else if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	} else if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}
	return ""
}
