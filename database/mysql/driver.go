package mysql

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nicolasbonnici/gorest/database"
)

type MySQLDriver struct {
	db           *sql.DB
	dialect      *MySQLDialect
	introspector *MySQLIntrospector
	poolCfg      database.PoolConfig
}

func init() {
	database.Register("mysql", func() database.Database {
		return &MySQLDriver{
			dialect: &MySQLDialect{},
		}
	})
}

func (d *MySQLDriver) ConfigurePool(cfg database.PoolConfig) {
	d.poolCfg = cfg
}

func (d *MySQLDriver) Connect(ctx context.Context, dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	if d.poolCfg.MaxOpen > 0 {
		db.SetMaxOpenConns(d.poolCfg.MaxOpen)
	}
	if d.poolCfg.MaxIdle > 0 {
		db.SetMaxIdleConns(d.poolCfg.MaxIdle)
	}
	if d.poolCfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(d.poolCfg.ConnMaxLifetime)
	}
	if d.poolCfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(d.poolCfg.ConnMaxIdleTime)
	}

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	d.db = db
	d.introspector = &MySQLIntrospector{db: db}
	return nil
}

func (d *MySQLDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *MySQLDriver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *MySQLDriver) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (d *MySQLDriver) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	row := d.db.QueryRowContext(ctx, query, args...)
	return &sqlRow{row: row}
}

func (d *MySQLDriver) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *MySQLDriver) Begin(ctx context.Context) (database.Tx, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

func (d *MySQLDriver) Dialect() database.Dialect {
	return d.dialect
}

func (d *MySQLDriver) DriverName() string {
	return "mysql"
}

func (d *MySQLDriver) Introspector() database.SchemaIntrospector {
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
