package mysql

import (
	"github.com/nicolasbonnici/gorest/database"
)

type MySQLDialect struct {
	database.BaseDialect
}

func (d *MySQLDialect) Placeholder(n int) string {
	return "?"
}

func (d *MySQLDialect) SupportsReturning() bool {
	return false
}

func (d *MySQLDialect) ReturningClause(cols ...string) string {
	return ""
}

func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

func (d *MySQLDialect) MapType(stdType string) string {
	switch database.StandardType(stdType) {
	case database.TypeInteger:
		return "INT"
	case database.TypeBigInt:
		return "BIGINT"
	case database.TypeString:
		return "VARCHAR(255)"
	case database.TypeText:
		return "TEXT"
	case database.TypeBoolean:
		return "TINYINT(1)"
	case database.TypeTimestamp:
		return "DATETIME"
	case database.TypeUUID:
		return "CHAR(36)"
	case database.TypeJSON:
		return "JSON"
	case database.TypeFloat:
		return "DOUBLE"
	case database.TypeDecimal:
		return "DECIMAL(10,2)"
	case database.TypeDate:
		return "DATE"
	case database.TypeTime:
		return "TIME"
	case database.TypeBytea:
		return "BLOB"
	default:
		return stdType
	}
}
