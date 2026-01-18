package database

import "fmt"

type Dialect interface {
	Placeholder(n int) string
	SupportsReturning() bool
	ReturningClause(cols ...string) string
	LimitOffset(limit, offset int) string
	QuoteIdentifier(name string) string
	MapType(dbType string) string
	CaseInsensitiveLike() string

	// Query builder capabilities
	SupportsFullJoin() bool
	SupportsWindowFunctions() bool
	SupportsCTE() bool
	SupportsArrays() bool
	OnConflictClause(columns []string, action string) string
	UpsertSupport() bool
}

type BaseDialect struct{}

func (d *BaseDialect) Placeholder(n int) string {
	return "?"
}

func (d *BaseDialect) SupportsReturning() bool {
	return false
}

func (d *BaseDialect) ReturningClause(cols ...string) string {
	return ""
}

func (d *BaseDialect) QuoteIdentifier(name string) string {
	return name
}

func (d *BaseDialect) MapType(dbType string) string {
	return dbType
}

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

func (d *BaseDialect) CaseInsensitiveLike() string {
	return "LOWER"
}

func (d *BaseDialect) SupportsFullJoin() bool {
	return false
}

func (d *BaseDialect) SupportsWindowFunctions() bool {
	return false
}

func (d *BaseDialect) SupportsCTE() bool {
	return false
}

func (d *BaseDialect) SupportsArrays() bool {
	return false
}

func (d *BaseDialect) OnConflictClause(columns []string, action string) string {
	return ""
}

func (d *BaseDialect) UpsertSupport() bool {
	return false
}
