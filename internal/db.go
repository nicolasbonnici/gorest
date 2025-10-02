package internal

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TableSchema struct {
	TableName string
	Columns   []string
	Relations []Relation
}

type Relation struct {
	ChildTable   string
	ChildColumn  string
	ParentTable  string
	ParentColumn string
}

func LoadSchema(db *pgxpool.Pool) map[string]TableSchema {
	tables := map[string]TableSchema{}

	colQuery := `
	SELECT table_name, column_name
	FROM information_schema.columns
	WHERE table_schema='public'
	ORDER BY table_name, ordinal_position;
	`
	rows, err := db.Query(context.Background(), colQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var table, col string
		if err := rows.Scan(&table, &col); err != nil {
			log.Fatal(err)
		}
		if _, ok := tables[table]; !ok {
			tables[table] = TableSchema{TableName: table, Columns: []string{}, Relations: []Relation{}}
		}
		ts := tables[table]
		ts.Columns = append(ts.Columns, col)
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
	relRows, err := db.Query(context.Background(), relQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer relRows.Close()

	for relRows.Next() {
		var r Relation
		if err := relRows.Scan(&r.ChildTable, &r.ChildColumn, &r.ParentTable, &r.ParentColumn); err != nil {
			log.Fatal(err)
		}
		ts := tables[r.ChildTable]
		ts.Relations = append(ts.Relations, r)
		tables[r.ChildTable] = ts
	}

	return tables
}
