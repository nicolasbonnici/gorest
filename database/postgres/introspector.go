package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/database"
)

type PostgresIntrospector struct {
	pool *pgxpool.Pool
}

func (i *PostgresIntrospector) LoadSchema(ctx context.Context) ([]database.TableSchema, error) {
	tables := make(map[string]database.TableSchema)

	colQuery := `
	SELECT table_name, column_name, is_nullable,
	       CASE
	           WHEN data_type = 'USER-DEFINED' THEN udt_name
	           ELSE data_type
	       END as data_type
	FROM information_schema.columns
	WHERE table_schema='public'
	ORDER BY table_name, ordinal_position;
	`

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := i.pool.Query(ctx, colQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var table, col, nullable, dataType string
		if err := rows.Scan(&table, &col, &nullable, &dataType); err != nil {
			return nil, err
		}

		if _, ok := tables[table]; !ok {
			tables[table] = database.TableSchema{
				TableName: table,
				Columns:   []database.Column{},
				Relations: []database.Relation{},
			}
		}

		ts := tables[table]
		ts.Columns = append(ts.Columns, database.Column{
			Name:       col,
			Type:       dataType,
			IsNullable: nullable == "YES",
		})
		tables[table] = ts
	}

	relQuery := `
	SELECT
		kcu.table_name AS child_table,
		kcu.column_name AS child_column,
		ccu.table_name AS parent_table,
		ccu.column_name AS parent_column
	FROM
		information_schema.key_column_usage kcu
	JOIN information_schema.constraint_column_usage ccu
		ON kcu.constraint_name = ccu.constraint_name
	WHERE kcu.table_schema='public';
	`

	relRows, err := i.pool.Query(ctx, relQuery)
	if err != nil {
		return nil, err
	}
	defer relRows.Close()

	for relRows.Next() {
		var r database.Relation
		if err := relRows.Scan(&r.ChildTable, &r.ChildColumn, &r.ParentTable, &r.ParentColumn); err != nil {
			return nil, err
		}
		ts := tables[r.ChildTable]
		ts.Relations = append(ts.Relations, r)
		tables[r.ChildTable] = ts
	}

	result := make([]database.TableSchema, 0, len(tables))
	for _, t := range tables {
		result = append(result, t)
	}

	return result, nil
}

func (i *PostgresIntrospector) GetColumns(ctx context.Context, tableName string) ([]database.Column, error) {
	query := `
	SELECT column_name,
	       CASE
	           WHEN data_type = 'USER-DEFINED' THEN udt_name
	           ELSE data_type
	       END as data_type,
	       is_nullable
	FROM information_schema.columns
	WHERE table_schema='public' AND table_name=$1
	ORDER BY ordinal_position;
	`

	rows, err := i.pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []database.Column
	for rows.Next() {
		var col, dataType, nullable string
		if err := rows.Scan(&col, &dataType, &nullable); err != nil {
			return nil, err
		}

		columns = append(columns, database.Column{
			Name:       col,
			Type:       dataType,
			IsNullable: nullable == "YES",
		})
	}

	return columns, nil
}

func (i *PostgresIntrospector) GetRelations(ctx context.Context) ([]database.Relation, error) {
	query := `
	SELECT
		kcu.table_name AS child_table,
		kcu.column_name AS child_column,
		ccu.table_name AS parent_table,
		ccu.column_name AS parent_column
	FROM
		information_schema.key_column_usage kcu
	JOIN information_schema.constraint_column_usage ccu
		ON kcu.constraint_name = ccu.constraint_name
	WHERE kcu.table_schema='public';
	`

	rows, err := i.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []database.Relation
	for rows.Next() {
		var r database.Relation
		if err := rows.Scan(&r.ChildTable, &r.ChildColumn, &r.ParentTable, &r.ParentColumn); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}

	return relations, nil
}
