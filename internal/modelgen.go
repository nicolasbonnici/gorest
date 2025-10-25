package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nicolasbonnici/gorest/pkg/database"
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

func LoadSchema(db database.Database) map[string]TableSchema {
	schemaSlice, err := db.Introspector().LoadSchema(context.Background())
	if err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	tables := make(map[string]TableSchema)
	for _, t := range schemaSlice {
		columns := make([]Column, len(t.Columns))
		for i, c := range t.Columns {
			columns[i] = Column{
				Name:       c.Name,
				Type:       c.Type,
				IsNullable: c.IsNullable,
			}
		}

		relations := make([]Relation, len(t.Relations))
		for i, r := range t.Relations {
			relations[i] = Relation{
				ChildTable:   r.ChildTable,
				ChildColumn:  r.ChildColumn,
				ParentTable:  r.ParentTable,
				ParentColumn: r.ParentColumn,
			}
		}

		tables[t.TableName] = TableSchema{
			TableName: t.TableName,
			Columns:   columns,
			Relations: relations,
		}
	}

	return tables
}

func GenerateStructs(tables map[string]TableSchema) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	modelsDir := fmt.Sprintf("%s/internal/models", projectRoot)
	os.MkdirAll(modelsDir, 0755)

	for _, table := range tables {
		singularTable := singularize(table.TableName)
		structName := toCamelCase(singularTable)
        switch structName {
        case "model":
            continue
        }

		filePath := fmt.Sprintf("%s/internal/models/%s.go", projectRoot, strings.ToLower(structName))

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

	apiDir := fmt.Sprintf("%s/internal/openapi", projectRoot)
	os.MkdirAll(apiDir, 0755)
	filePath := fmt.Sprintf("%s/internal/openapi/openapi_gen.go", projectRoot)

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
	fmt.Println("✅ Generated OpenAPI resource stubs → internal/openapi/openapi_gen.go")
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

func singularize(word string) string {
	return SingularizeExported(word)
}

func SingularizeExported(word string) string {
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ves") {
		if len(word) > 4 && word[len(word)-4] == 'l' {
			return word[:len(word)-3] + "f"
		}
		return word[:len(word)-3] + "fe"
	}
	if strings.HasSuffix(word, "ses") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "xes") || strings.HasSuffix(word, "zes") ||
	   strings.HasSuffix(word, "ches") || strings.HasSuffix(word, "shes") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		return word[:len(word)-1]
	}
	return word
}

func ScaffoldAll(db database.Database) {
	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateAPI(db, tables, NoAuthConfig())
	GenerateOpenAPI(tables)
}
