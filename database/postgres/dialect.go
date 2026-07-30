package postgres

import (
	"fmt"
	"strings"

	"github.com/nicolasbonnici/gorest/database"
)

type PostgresDialect struct {
	database.BaseDialect
}

func (d *PostgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (d *PostgresDialect) SupportsReturning() bool {
	return true
}

func (d *PostgresDialect) ReturningClause(cols ...string) string {
	if len(cols) == 0 {
		return "RETURNING id"
	}
	quoted := make([]string, len(cols))
	for i, col := range cols {
		quoted[i] = d.QuoteIdentifier(col)
	}
	return "RETURNING " + strings.Join(quoted, ", ")
}

func (d *PostgresDialect) QuoteIdentifier(name string) string {
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return `"` + escaped + `"`
}

func (d *PostgresDialect) MapType(stdType string) string {
	switch database.StandardType(stdType) {
	case database.TypeInteger:
		return "INTEGER"
	case database.TypeBigInt:
		return "BIGINT"
	case database.TypeString:
		return "VARCHAR(255)"
	case database.TypeText:
		return "TEXT"
	case database.TypeBoolean:
		return "BOOLEAN"
	case database.TypeTimestamp:
		return "TIMESTAMP WITH TIME ZONE"
	case database.TypeUUID:
		return "UUID"
	case database.TypeJSON:
		return "JSONB"
	case database.TypeFloat:
		return "DOUBLE PRECISION"
	case database.TypeDecimal:
		return "NUMERIC"
	case database.TypeDate:
		return "DATE"
	case database.TypeTime:
		return "TIME"
	case database.TypeBytea:
		return "BYTEA"
	default:
		return stdType
	}
}

func (d *PostgresDialect) CaseInsensitiveLike() string {
	return "ILIKE"
}

func (d *PostgresDialect) SupportsFullJoin() bool {
	return true
}

func (d *PostgresDialect) SupportsWindowFunctions() bool {
	return true
}

func (d *PostgresDialect) SupportsCTE() bool {
	return true
}

func (d *PostgresDialect) SupportsArrays() bool {
	return true
}

func (d *PostgresDialect) OnConflictClause(columns []string, action string) string {
	if len(columns) == 0 {
		return ""
	}

	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = d.QuoteIdentifier(col)
	}

	conflict := fmt.Sprintf("ON CONFLICT (%s)", strings.Join(quotedColumns, ", "))

	if action != "" {
		conflict += " " + action
	}

	return conflict
}

func (d *PostgresDialect) UpsertSupport() bool {
	return true
}

// EstimateRowsQuery reads the planner's row estimate from pg_class. reltuples is
// -1 on tables that have never been analyzed, which callers treat as unknown.
func (d *PostgresDialect) EstimateRowsQuery(table string) (string, []any, bool) {
	return "SELECT reltuples::bigint FROM pg_class WHERE oid = to_regclass($1)", []any{table}, true
}
