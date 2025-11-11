package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicolasbonnici/gorest/database"
)

type SQLiteIntrospector struct {
	db *sql.DB
}

func (i *SQLiteIntrospector) LoadSchema(ctx context.Context) ([]database.TableSchema, error) {
	tableQuery := `
	SELECT name FROM sqlite_master
	WHERE type='table' AND name NOT LIKE 'sqlite_%'
	ORDER BY name;
	`

	rows, err := i.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}

	var result []database.TableSchema
	for _, tableName := range tableNames {
		columns, err := i.GetColumns(ctx, tableName)
		if err != nil {
			return nil, err
		}

		relations, err := i.getTableRelations(ctx, tableName)
		if err != nil {
			return nil, err
		}

		result = append(result, database.TableSchema{
			TableName: tableName,
			Columns:   columns,
			Relations: relations,
		})
	}

	return result, nil
}

func (i *SQLiteIntrospector) GetColumns(ctx context.Context, tableName string) ([]database.Column, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s);", tableName)

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []database.Column
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}

		columns = append(columns, database.Column{
			Name:       name,
			Type:       dataType,
			IsNullable: notNull == 0,
		})
	}

	return columns, nil
}

func (i *SQLiteIntrospector) GetRelations(ctx context.Context) ([]database.Relation, error) {
	tableQuery := `
	SELECT name FROM sqlite_master
	WHERE type='table' AND name NOT LIKE 'sqlite_%'
	ORDER BY name;
	`

	rows, err := i.db.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}

	var relations []database.Relation
	for _, tableName := range tableNames {
		tableRelations, err := i.getTableRelations(ctx, tableName)
		if err != nil {
			return nil, err
		}
		relations = append(relations, tableRelations...)
	}

	return relations, nil
}

func (i *SQLiteIntrospector) getTableRelations(ctx context.Context, tableName string) ([]database.Relation, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s);", tableName)

	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []database.Relation
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		relations = append(relations, database.Relation{
			ChildTable:   tableName,
			ChildColumn:  from,
			ParentTable:  table,
			ParentColumn: to,
		})
	}

	return relations, nil
}
