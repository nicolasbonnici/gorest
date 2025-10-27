package sqlite

import (
	"github.com/nicolasbonnici/gorest/pkg/database"
)

type SQLiteDialect struct {
	database.BaseDialect
}

func (d *SQLiteDialect) Placeholder(n int) string {
	return "?"
}

func (d *SQLiteDialect) SupportsReturning() bool {
	return true
}

func (d *SQLiteDialect) ReturningClause(cols ...string) string {
	if len(cols) == 0 {
		return "RETURNING id"
	}
	return "RETURNING " + d.QuoteIdentifier(cols[0])
}

func (d *SQLiteDialect) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}

func (d *SQLiteDialect) MapType(stdType string) string {
	switch database.StandardType(stdType) {
	case database.TypeInteger:
		return "INTEGER"
	case database.TypeBigInt:
		return "INTEGER"
	case database.TypeString, database.TypeText, database.TypeUUID:
		return "TEXT"
	case database.TypeBoolean:
		return "INTEGER"
	case database.TypeTimestamp, database.TypeDate, database.TypeTime:
		return "TEXT"
	case database.TypeJSON:
		return "TEXT"
	case database.TypeFloat, database.TypeDecimal:
		return "REAL"
	case database.TypeBytea:
		return "BLOB"
	default:
		return stdType
	}
}
