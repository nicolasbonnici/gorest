package database

import (
	"testing"
)

func TestDetectDriver(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		expected string
	}{
		{"postgres with postgres://", "postgres://user:pass@localhost/db", "postgres"},
		{"postgres with postgresql://", "postgresql://user:pass@localhost/db", "postgres"},
		{"mysql with mysql://", "mysql://user:pass@localhost/db", "mysql"},
		{"mysql with @tcp", "user:pass@tcp(localhost:3306)/db", "mysql"},
		{"sqlite with file:", "file:./test.db", "sqlite"},
		{"sqlite with .db", "./test.db", "sqlite"},
		{"sqlite with .sqlite", "./test.sqlite", "sqlite"},
		{"sqlite with .sqlite3", "./test.sqlite3", "sqlite"},
		{"default to postgres", "something:else", "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectDriver(tt.dsn)
			if result != tt.expected {
				t.Errorf("DetectDriver(%q) = %q, want %q", tt.dsn, result, tt.expected)
			}
		})
	}
}

func TestStandardizeType(t *testing.T) {
	tests := []struct {
		dbType     string
		driverName string
		expected   StandardType
	}{
		{"integer", "postgres", TypeInteger},
		{"bigint", "postgres", TypeBigInt},
		{"text", "postgres", TypeText},
		{"boolean", "postgres", TypeBoolean},
		{"uuid", "postgres", TypeUUID},
		{"jsonb", "postgres", TypeJSON},
		{"timestamp with time zone", "postgres", TypeTimestamp},

		{"int", "mysql", TypeInteger},
		{"bigint", "mysql", TypeBigInt},
		{"varchar", "mysql", TypeText},
		{"tinyint", "mysql", TypeBoolean},
		{"json", "mysql", TypeJSON},
		{"datetime", "mysql", TypeTimestamp},

		{"INTEGER", "sqlite", TypeInteger},
		{"TEXT", "sqlite", TypeText},
		{"REAL", "sqlite", TypeFloat},
		{"BLOB", "sqlite", TypeBytea},
		{"DATETIME", "sqlite", TypeTimestamp},

		{"unknown", "postgres", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.dbType+"_"+tt.driverName, func(t *testing.T) {
			result := StandardizeType(tt.dbType, tt.driverName)
			if result != tt.expected {
				t.Errorf("StandardizeType(%q, %q) = %q, want %q", tt.dbType, tt.driverName, result, tt.expected)
			}
		})
	}
}
