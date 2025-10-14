package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, colQuery)
	if err != nil {
		log.Fatalf("Failed to query database columns: %v", err)
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

	relCtx, relCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer relCancel()

	relRows, err := db.Query(relCtx, relQuery)
	if err != nil {
		log.Fatalf("Failed to query table relations: %v", err)
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

	modelsDir := fmt.Sprintf("%s/internal/api/models", projectRoot)
	os.MkdirAll(modelsDir, 0755)

	for _, table := range tables {
		singularTable := singularize(table.TableName)
		structName := toCamelCase(singularTable)
        switch structName {
        case "model":
            continue
        }

		filePath := fmt.Sprintf("%s/internal/api/models/%s.go", projectRoot, strings.ToLower(structName))

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

		if err := os.WriteFile(filePath, []byte(b.String()), 0644); err != nil {
			log.Fatalf("Failed to write file %s: %v", filePath, err)
		}
		fmt.Printf("✅ Generated struct for table: %s → %s\n", table.TableName, filePath)
	}
}

func GenerateOpenAPI(tables map[string]TableSchema) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	apiDir := fmt.Sprintf("%s/internal/api/openapi", projectRoot)
	os.MkdirAll(apiDir, 0755)
	filePath := fmt.Sprintf("%s/internal/api/openapi/openapi_gen.go", projectRoot)

	var b strings.Builder
	b.WriteString("package api\n\n")
	b.WriteString("// Auto-generated OpenAPI schema stubs\n\n")

	for _, table := range tables {
		singularTable := singularize(table.TableName)
		resource := toCamelCase(singularTable)
		b.WriteString(fmt.Sprintf("// %sResource defines OpenAPI schema and endpoints for %s\n", resource, table.TableName))
		b.WriteString(fmt.Sprintf("type %sResource struct {}\n\n", resource))
	}

	os.WriteFile(filePath, []byte(b.String()), 0644)
	fmt.Println("✅ Generated OpenAPI resource stubs → internal/api/openapi/openapi_gen.go")
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
	caser := cases.Title(language.English)
	for i, p := range parts {
		parts[i] = caser.String(p)
	}
	return strings.Join(parts, "")
}

// singularize converts a plural word to singular
func singularize(word string) string {
	return SingularizeExported(word)
}

// SingularizeExported converts a plural word to singular (exported for use in other packages)
func SingularizeExported(word string) string {
	// Handle common plural patterns
	if strings.HasSuffix(word, "ies") {
		// categories -> category, stories -> story
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ves") {
		// knives -> knife, wolves -> wolf
		// Check if it ends in "lves" (wolves, shelves) -> keep 'f'
		if len(word) > 4 && word[len(word)-4] == 'l' {
			return word[:len(word)-3] + "f"
		}
		return word[:len(word)-3] + "fe"
	}
	if strings.HasSuffix(word, "ses") {
		// classes -> class, addresses -> address
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "xes") || strings.HasSuffix(word, "zes") ||
	   strings.HasSuffix(word, "ches") || strings.HasSuffix(word, "shes") {
		// boxes -> box, buzzes -> buzz, churches -> church, dishes -> dish
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		// users -> user, todos -> todo
		// but keep "address", "process", etc.
		return word[:len(word)-1]
	}
	return word
}

func ScaffoldAll(db *pgxpool.Pool) {
	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateAPI(db, tables, NoAuthConfig())
	GenerateOpenAPI(tables)
}
