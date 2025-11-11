package database

import (
	"context"
	"fmt"
	"strings"
)

type DriverFactory func() Database

var drivers = make(map[string]DriverFactory)

func Register(name string, factory DriverFactory) {
	drivers[name] = factory
}

func Open(driverName, dsn string) (Database, error) {
	if driverName == "" {
		driverName = DetectDriver(dsn)
	}

	factory, ok := drivers[driverName]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", driverName)
	}

	db := factory()
	if err := db.Connect(context.Background(), dsn); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return db, nil
}

func DetectDriver(dsn string) string {
	dsn = strings.ToLower(dsn)

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return "postgres"
	}

	if strings.HasPrefix(dsn, "mysql://") || strings.Contains(dsn, "@tcp(") {
		return "mysql"
	}

	if strings.HasPrefix(dsn, "file:") || strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite") || strings.HasSuffix(dsn, ".sqlite3") {
		return "sqlite"
	}

	return "postgres"
}
