package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/database"
)

type PostgresDriver struct {
	pool         *pgxpool.Pool
	dialect      *PostgresDialect
	introspector *PostgresIntrospector
}

func init() {
	database.Register("postgres", func() database.Database {
		return &PostgresDriver{
			dialect: &PostgresDialect{},
		}
	})
}

func (d *PostgresDriver) Connect(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}

	d.pool = pool
	d.introspector = &PostgresIntrospector{pool: pool}
	return nil
}

func (d *PostgresDriver) Close() error {
	if d.pool != nil {
		d.pool.Close()
	}
	return nil
}

func (d *PostgresDriver) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *PostgresDriver) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

func (d *PostgresDriver) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	row := d.pool.QueryRow(ctx, query, args...)
	return &pgxRow{row: row}
}

func (d *PostgresDriver) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	tag, err := d.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{tag: tag}, nil
}

func (d *PostgresDriver) Begin(ctx context.Context) (database.Tx, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{tx: tx}, nil
}

func (d *PostgresDriver) Dialect() database.Dialect {
	return d.dialect
}

func (d *PostgresDriver) DriverName() string {
	return "postgres"
}

func (d *PostgresDriver) Introspector() database.SchemaIntrospector {
	return d.introspector
}

type pgxRows struct {
	rows pgx.Rows
}

func (r *pgxRows) Next() bool {
	return r.rows.Next()
}

func (r *pgxRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *pgxRows) Close() error {
	r.rows.Close()
	return nil
}

func (r *pgxRows) Err() error {
	return r.rows.Err()
}

type pgxRow struct {
	row pgx.Row
}

func (r *pgxRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

type pgxResult struct {
	tag pgconn.CommandTag
}

func (r *pgxResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *pgxResult) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}

type pgxTx struct {
	tx pgx.Tx
}

func (t *pgxTx) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

func (t *pgxTx) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	row := t.tx.QueryRow(ctx, query, args...)
	return &pgxRow{row: row}
}

func (t *pgxTx) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	tag, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{tag: tag}, nil
}

func (t *pgxTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *pgxTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
