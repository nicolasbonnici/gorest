package database

import (
	"context"
)

type Database interface {
	Connect(ctx context.Context, dsn string) error
	Close() error
	Ping(ctx context.Context) error

	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)

	Begin(ctx context.Context) (Tx, error)

	Dialect() Dialect
	DriverName() string
	Introspector() SchemaIntrospector
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

type Row interface {
	Scan(dest ...interface{}) error
}

type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type Tx interface {
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
