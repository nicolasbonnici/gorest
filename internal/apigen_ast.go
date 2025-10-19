package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

// StructField represents a field in a Go struct
type StructField struct {
	Name      string
	Type      string
	JSONTag   string
	DBTag     string
	DTOTag    string // Controls field inclusion in DTOs: "read", "write", "read,write", or "-"
	IsPointer bool
}

// parseStructs extracts struct names from a Go source file
func parseStructs(path string) []string {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
	if err != nil {
		log.Printf("parse error in %s: %v", path, err)
		return nil
	}

	structs := []string{}
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := ts.Type.(*ast.StructType); ok {
				structs = append(structs, ts.Name.Name)
			}
		}
	}
	return structs
}

// extractStructFields extracts field information from a struct definition
func extractStructFields(path string, structName string) []StructField {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
	if err != nil {
		log.Printf("parse error in %s: %v", path, err)
		return nil
	}

	var fields []StructField
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != structName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				fieldName := field.Names[0].Name
				fieldType := ""
				isPointer := false

				switch t := field.Type.(type) {
				case *ast.Ident:
					fieldType = t.Name
				case *ast.StarExpr:
					isPointer = true
					if ident, ok := t.X.(*ast.Ident); ok {
						fieldType = ident.Name
					} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
						if pkg, ok := sel.X.(*ast.Ident); ok {
							fieldType = pkg.Name + "." + sel.Sel.Name
						}
					}
				case *ast.SelectorExpr:
					if pkg, ok := t.X.(*ast.Ident); ok {
						fieldType = pkg.Name + "." + t.Sel.Name
					}
				}

				jsonTag := ""
				dbTag := ""
				dtoTag := ""
				if field.Tag != nil {
					tag := field.Tag.Value
					jsonTag = extractTag(tag, "json")
					dbTag = extractTag(tag, "db")
					dtoTag = extractTag(tag, "dto")
				}

				fields = append(fields, StructField{
					Name:      fieldName,
					Type:      fieldType,
					JSONTag:   jsonTag,
					DBTag:     dbTag,
					DTOTag:    dtoTag,
					IsPointer: isPointer,
				})
			}
		}
	}
	return fields
}

// extractTag extracts a tag value from a struct tag string
func extractTag(tagString, key string) string {
	tagString = strings.Trim(tagString, "`")
	for _, tag := range strings.Fields(tagString) {
		if strings.HasPrefix(tag, key+":") {
			value := strings.TrimPrefix(tag, key+":")
			value = strings.Trim(value, `"`)
			return value
		}
	}
	return ""
}

// extractStructFieldsFromAST extracts fields from an AST StructType
func extractStructFieldsFromAST(st *ast.StructType) []StructField {
	var fields []StructField

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		fieldType := ""
		isPointer := false

		switch t := field.Type.(type) {
		case *ast.Ident:
			fieldType = t.Name
		case *ast.StarExpr:
			isPointer = true
			if ident, ok := t.X.(*ast.Ident); ok {
				fieldType = ident.Name
			} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
				if pkg, ok := sel.X.(*ast.Ident); ok {
					fieldType = pkg.Name + "." + sel.Sel.Name
				}
			}
		case *ast.SelectorExpr:
			if pkg, ok := t.X.(*ast.Ident); ok {
				fieldType = pkg.Name + "." + t.Sel.Name
			}
		}

		jsonTag := ""
		dbTag := ""
		dtoTag := ""
		if field.Tag != nil {
			tag := field.Tag.Value
			jsonTag = extractTag(tag, "json")
			jsonTag = strings.Split(jsonTag, ",")[0]
			dbTag = extractTag(tag, "db")
			dtoTag = extractTag(tag, "dto")
		}

		fields = append(fields, StructField{
			Name:      fieldName,
			Type:      fieldType,
			JSONTag:   jsonTag,
			DBTag:     dbTag,
			DTOTag:    dtoTag,
			IsPointer: isPointer,
		})
	}

	return fields
}

// extractDTOsFromResourceFile extracts DTO schemas from a resource file
func extractDTOsFromResourceFile(path string) map[string]DTOSchema {
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
	if err != nil {
		log.Printf("parse error in %s: %v", path, err)
		return nil
	}

	dtos := make(map[string]DTOSchema)

	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if !strings.HasSuffix(ts.Name.Name, "DTO") {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			fields := extractStructFieldsFromAST(st)
			dtos[ts.Name.Name] = DTOSchema{
				Name:   ts.Name.Name,
				Fields: fields,
			}
		}
	}

	return dtos
}
