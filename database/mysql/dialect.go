package mysql

import (
	"fmt"
	"strings"

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
	escaped := strings.ReplaceAll(name, "`", "``")
	return "`" + escaped + "`"
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

func (d *MySQLDialect) SupportsFullJoin() bool {
	return false
}

func (d *MySQLDialect) SupportsWindowFunctions() bool {
	return true
}

func (d *MySQLDialect) SupportsCTE() bool {
	return true
}

func (d *MySQLDialect) SupportsArrays() bool {
	return false
}

func (d *MySQLDialect) OnConflictClause(columns []string, action string) string {
	if len(columns) == 0 {
		return ""
	}

	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = d.QuoteIdentifier(col)
	}

	updateParts := make([]string, len(columns))
	for i, col := range columns {
		quotedCol := d.QuoteIdentifier(col)
		updateParts[i] = fmt.Sprintf("%s = VALUES(%s)", quotedCol, quotedCol)
	}

	if action == "DO NOTHING" {
		return ""
	}

	return "ON DUPLICATE KEY UPDATE " + strings.Join(updateParts, ", ")
}

func (d *MySQLDialect) UpsertSupport() bool {
	return true
}

// EstimateRowsQuery reads TABLE_ROWS from information_schema. It is an estimate
// derived from index statistics on InnoDB, and NULL for a table the current
// schema does not own, which maps to -1 so callers treat it as unknown.
func (d *MySQLDialect) EstimateRowsQuery(table string) (string, []any, bool) {
	return "SELECT COALESCE(TABLE_ROWS, -1) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", []any{table}, true
}
