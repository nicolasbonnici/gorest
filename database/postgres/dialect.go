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
