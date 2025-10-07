package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TableSchema struct {
	TableName string
	Columns   []Column
	Relations []Relation
}

type Column struct {
	Name       string
	Type       string
	IsNullable bool
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
	SELECT table_name, column_name, is_nullable, data_type
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
		var table, col, nullable, dataType string
		if err := rows.Scan(&table, &col, &nullable, &dataType); err != nil {
			log.Fatal(err)
		}
		if _, ok := tables[table]; !ok {
			tables[table] = TableSchema{TableName: table, Columns: []Column{}, Relations: []Relation{}}
		}
		ts := tables[table]
		ts.Columns = append(ts.Columns, Column{
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

func GenerateStructs(tables map[string]TableSchema) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	modelsDir := fmt.Sprintf("%s/gen/models", projectRoot)
	os.MkdirAll(modelsDir, 0755)

	for _, table := range tables {
		structName := toCamelCase(table.TableName)
        switch structName {
        case "model":
            continue
        }

		filePath := fmt.Sprintf("%s/gen/models/%s.go", projectRoot, strings.ToLower(structName))

		needsTime := false
		for _, col := range table.Columns {
			if strings.Contains(col.Type, "timestamp") {
				needsTime = true
				break
			}
		}

		var b strings.Builder
		b.WriteString("package models\n\n")
		if needsTime {
			b.WriteString("import \"time\"\n\n")
		}
		b.WriteString("type " + structName + " struct {\n")

		for _, col := range table.Columns {
			fieldName := toCamelCase(col.Name)
			fieldType := pgToGoType(col.Type, col.IsNullable)

			omitempty := ""
			if col.Name == "id" || col.Name == "created_at" || col.Name == "updated_at" || col.IsNullable {
				omitempty = ",omitempty"
			}

			jsonTag := fmt.Sprintf("`json:\"%s%s\" db:\"%s\"`", col.Name, omitempty, col.Name)
			b.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag))
		}
		b.WriteString("}\n")
		b.WriteString("\n")
		b.WriteString("func (" + structName + ") TableName() string {\n")
		b.WriteString("	return \"" + table.TableName + "\" \n")
		b.WriteString("}\n")

		os.WriteFile(filePath, []byte(b.String()), 0644)
		fmt.Printf("✅ Generated struct for table: %s → %s\n", table.TableName, filePath)
	}
}

func GenerateOpenAPI(tables map[string]TableSchema) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	apiDir := fmt.Sprintf("%s/gen/api", projectRoot)
	os.MkdirAll(apiDir, 0755)
	filePath := fmt.Sprintf("%s/gen/api/openapi_gen.go", projectRoot)

	var b strings.Builder
	b.WriteString("package api\n\n")
	b.WriteString("// Auto-generated OpenAPI schema stubs\n\n")

	for _, table := range tables {
		resource := toCamelCase(table.TableName)
		b.WriteString(fmt.Sprintf("// %sResource defines OpenAPI schema and endpoints for %s\n", resource, table.TableName))
		b.WriteString(fmt.Sprintf("type %sResource struct {}\n\n", resource))
	}

	os.WriteFile(filePath, []byte(b.String()), 0644)
	fmt.Println("✅ Generated OpenAPI resource stubs → gen/api/openapi_gen.go")
}

func pgToGoType(pgType string, nullable bool) string {
	base := map[string]string{
		"integer":  "int",
		"bigint":   "int64",
		"smallint": "int16",
		"text":     "string",
		"varchar":  "string",
		"character varying": "string",
		"boolean":  "bool",
		"timestamp without time zone": "time.Time",
		"timestamp with time zone": "time.Time",
		"timestamp": "time.Time",
		"uuid": "string",
		"numeric": "float64",
		"double precision": "float64",
		"real": "float32",
		"json": "map[string]interface{}",
		"jsonb": "map[string]interface{}",
	}
	goType, ok := base[pgType]
	if !ok {
		goType = "interface{}"
	}

	isTimestamp := goType == "time.Time"
	if (nullable || isTimestamp) && goType != "interface{}" {
		goType = "*" + goType
	}
	return goType
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}

func ScaffoldAll(db *pgxpool.Pool) {
	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateOpenAPI(tables)
}
