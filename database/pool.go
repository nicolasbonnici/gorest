package database

import (
	"context"
	"fmt"
	"time"
)

// PoolConfig holds connection pool settings shared across all drivers.
// Zero values are ignored so callers only need to set what they care about.
type PoolConfig struct {
	// MaxOpen is the maximum number of open connections (0 = driver default).
	MaxOpen int
	// MaxIdle is the maximum number of idle connections kept in the pool
	// (0 = driver default; maps to MinConns for pgxpool).
	MaxIdle int
	// ConnMaxLifetime is the maximum time a connection may be reused (0 = no limit).
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum time a connection may remain idle (0 = no limit).
	ConnMaxIdleTime time.Duration
}

// PoolConfigurer is implemented by drivers that support runtime pool tuning.
type PoolConfigurer interface {
	ConfigurePool(cfg PoolConfig)
}

// OpenWithPool opens a database connection and applies pool settings before
// the first connection is established. Drivers that do not implement
// PoolConfigurer silently ignore the config.
func OpenWithPool(driverName, dsn string, cfg PoolConfig) (Database, error) {
	if driverName == "" {
		driverName = DetectDriver(dsn)
	}
	factory, ok := drivers[driverName]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s", driverName)
	}
	db := factory()
	if pc, ok := db.(PoolConfigurer); ok {
		pc.ConfigurePool(cfg)
	}
	if err := db.Connect(context.Background(), dsn); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return db, nil
}
