package database

type StandardType string

const (
	TypeInteger   StandardType = "integer"
	TypeBigInt    StandardType = "bigint"
	TypeString    StandardType = "string"
	TypeText      StandardType = "text"
	TypeBoolean   StandardType = "boolean"
	TypeTimestamp StandardType = "timestamp"
	TypeUUID      StandardType = "uuid"
	TypeJSON      StandardType = "json"
	TypeFloat     StandardType = "float"
	TypeDecimal   StandardType = "decimal"
	TypeDate      StandardType = "date"
	TypeTime      StandardType = "time"
	TypeBytea     StandardType = "bytea"
)

func StandardizeType(dbType, driverName string) StandardType {
	switch driverName {
	case "postgres":
		return standardizePostgresType(dbType)
	case "mysql":
		return standardizeMySQLType(dbType)
	case "sqlite":
		return standardizeSQLiteType(dbType)
	default:
		return StandardType(dbType)
	}
}

func standardizePostgresType(dbType string) StandardType {
	switch dbType {
	case "integer", "int", "int4":
		return TypeInteger
	case "bigint", "int8":
		return TypeBigInt
	case "character varying", "varchar", "text":
		return TypeText
	case "boolean", "bool":
		return TypeBoolean
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return TypeTimestamp
	case "uuid":
		return TypeUUID
	case "json", "jsonb":
		return TypeJSON
	case "real", "float4", "double precision", "float8":
		return TypeFloat
	case "numeric", "decimal":
		return TypeDecimal
	case "date":
		return TypeDate
	case "time", "time without time zone", "time with time zone":
		return TypeTime
	case "bytea":
		return TypeBytea
	default:
		return StandardType(dbType)
	}
}

func standardizeMySQLType(dbType string) StandardType {
	switch dbType {
	case "int", "integer", "mediumint":
		return TypeInteger
	case "bigint":
		return TypeBigInt
	case "varchar", "text", "tinytext", "mediumtext", "longtext", "char":
		return TypeText
	case "tinyint":
		return TypeBoolean
	case "datetime", "timestamp":
		return TypeTimestamp
	case "json":
		return TypeJSON
	case "float", "double":
		return TypeFloat
	case "decimal", "numeric":
		return TypeDecimal
	case "date":
		return TypeDate
	case "time":
		return TypeTime
	case "blob", "varbinary", "binary":
		return TypeBytea
	default:
		return StandardType(dbType)
	}
}

func standardizeSQLiteType(dbType string) StandardType {
	switch dbType {
	case "INTEGER":
		return TypeInteger
	case "TEXT":
		return TypeText
	case "REAL":
		return TypeFloat
	case "BLOB":
		return TypeBytea
	case "DATETIME":
		return TypeTimestamp
	default:
		return StandardType(dbType)
	}
}
