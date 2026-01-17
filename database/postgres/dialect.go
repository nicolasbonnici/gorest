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
	return "RETURNING " + strings.Join(cols, ", ")
}

func (d *PostgresDialect) QuoteIdentifier(name string) string {
	return `"` + name + `"`
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
