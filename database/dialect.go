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

// RowEstimator is an optional capability a Dialect may implement to answer
// "roughly how many rows does this table hold?" from catalog statistics instead
// of a full COUNT(*). It is kept out of Dialect so third-party dialects keep
// compiling; callers type-assert for it and fall back to an exact count.
type RowEstimator interface {
	// EstimateRowsQuery returns a query yielding an approximate row count for
	// table. ok is false when the dialect has no cheap estimate available.
	//
	// The result may be stale, and negative when the database has never
	// gathered statistics for the table; callers must treat a negative result
	// as "unknown" rather than as a count.
	EstimateRowsQuery(table string) (sql string, args []any, ok bool)
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
