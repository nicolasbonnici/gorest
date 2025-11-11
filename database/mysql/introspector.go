package mysql

import (
	"context"
	"database/sql"

	"github.com/nicolasbonnici/gorest/database"
)

type MySQLIntrospector struct {
	db *sql.DB
}

func (i *MySQLIntrospector) LoadSchema(ctx context.Context) ([]database.TableSchema, error) {
	tables := make(map[string]database.TableSchema)

	colQuery := `
	SELECT table_name, column_name, is_nullable, data_type
	FROM information_schema.columns
	WHERE table_schema = DATABASE()
	ORDER BY table_name, ordinal_position;
	`

	rows, err := i.db.QueryContext(ctx, colQuery)
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
		kcu.referenced_table_name AS parent_table,
		kcu.referenced_column_name AS parent_column
	FROM
		information_schema.key_column_usage kcu
	WHERE
		kcu.table_schema = DATABASE()
		AND kcu.referenced_table_name IS NOT NULL;
	`

	relRows, err := i.db.QueryContext(ctx, relQuery)
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

func (i *MySQLIntrospector) GetColumns(ctx context.Context, tableName string) ([]database.Column, error) {
	query := `
	SELECT column_name, data_type, is_nullable
	FROM information_schema.columns
	WHERE table_schema = DATABASE() AND table_name = ?
	ORDER BY ordinal_position;
	`

	rows, err := i.db.QueryContext(ctx, query, tableName)
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

func (i *MySQLIntrospector) GetRelations(ctx context.Context) ([]database.Relation, error) {
	query := `
	SELECT
		kcu.table_name AS child_table,
		kcu.column_name AS child_column,
		kcu.referenced_table_name AS parent_table,
		kcu.referenced_column_name AS parent_column
	FROM
		information_schema.key_column_usage kcu
	WHERE
		kcu.table_schema = DATABASE()
		AND kcu.referenced_table_name IS NOT NULL;
	`

	rows, err := i.db.QueryContext(ctx, query)
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
