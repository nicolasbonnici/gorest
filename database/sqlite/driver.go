package sqlite

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
	"github.com/nicolasbonnici/gorest/database"
)

type SQLiteDriver struct {
	db           *sql.DB
	dialect      *SQLiteDialect
	introspector *SQLiteIntrospector
}

func init() {
	database.Register("sqlite", func() database.Database {
		return &SQLiteDriver{
			dialect: &SQLiteDialect{},
		}
	})
}

func (d *SQLiteDriver) Connect(ctx context.Context, dsn string) error {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	d.db = db
	d.introspector = &SQLiteIntrospector{db: db}
	return nil
}

func (d *SQLiteDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *SQLiteDriver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *SQLiteDriver) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (d *SQLiteDriver) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	row := d.db.QueryRowContext(ctx, query, args...)
	return &sqlRow{row: row}
}

func (d *SQLiteDriver) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *SQLiteDriver) Begin(ctx context.Context) (database.Tx, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

func (d *SQLiteDriver) Dialect() database.Dialect {
	return d.dialect
}

func (d *SQLiteDriver) DriverName() string {
	return "sqlite"
}

func (d *SQLiteDriver) Introspector() database.SchemaIntrospector {
	return d.introspector
}

type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Next() bool {
	return r.rows.Next()
}

func (r *sqlRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRows) Close() error {
	return r.rows.Close()
}

func (r *sqlRows) Err() error {
	return r.rows.Err()
}

type sqlRow struct {
	row *sql.Row
}

func (r *sqlRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

type sqlTx struct {
	tx *sql.Tx
}

func (t *sqlTx) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (t *sqlTx) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	row := t.tx.QueryRowContext(ctx, query, args...)
	return &sqlRow{row: row}
}

func (t *sqlTx) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *sqlTx) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *sqlTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}
