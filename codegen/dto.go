package codegen

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func toJSONCamelCase(s string) string {
	if s == "" {
		return s
	}

	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		caser := cases.Title(language.English)
		result := strings.ToLower(parts[0])
		for i := 1; i < len(parts); i++ {
			result += caser.String(parts[i])
		}
		return result
	}

	return strings.ToLower(s[:1]) + s[1:]
}

type DTOSchema struct {
	Name   string
	Fields []StructField
}

type ResourceDTOs struct {
	Name       string
	PluralName string
	DTOs       map[string]DTOSchema
}

func generateDTOForStruct(dtosDir string, structName string) {
	dtoFile := filepath.Join(dtosDir, strings.ToLower(structName)+".go")

	cfg, _ := LoadConfig()
	projectRoot, _ := findProjectRoot()
	modelsDir := cfg.Codegen.Output.Models
	if !filepath.IsAbs(modelsDir) {
		modelsDir = filepath.Join(projectRoot, modelsDir)
	}
	modelPath := filepath.Join(modelsDir, strings.ToLower(structName)+".go")
	fields := extractStructFields(modelPath, structName)

	code := generateDTOsFromModel(structName, fields)
	if err := os.WriteFile(dtoFile, []byte(code), 0644); err != nil {
		log.Fatalf("failed to write DTOs for %s: %v", structName, err)
	}
	log.Printf("📝 Generated DTOs for model: %s → %s", structName, dtoFile)
}

func generateDTOsFromModel(structName string, fields []StructField) string {
	needsTimeImport := false
	for _, f := range fields {
		if f.Type == "time.Time" {
			needsTimeImport = true
			break
		}
	}

	timeImport := ""
	if needsTimeImport {
		timeImport = `import "time"`
	}

	dtoFields := generateDTOFields(fields)
	createFields := generateCreateDTOFields(fields)
	updateFields := generateUpdateDTOFields(fields)

	return fmt.Sprintf(`package dtos

%s

type %sDTO struct {
%s}

type %sCreateDTO struct {
%s}

type %sUpdateDTO struct {
%s}
`, timeImport, structName, dtoFields, structName, createFields, structName, updateFields)
}

func generateDTOFields(fields []StructField) string {
	var result strings.Builder
	for _, field := range fields {
		if field.DTOTag == "-" || field.DTOTag == "write" {
			continue
		}

		typeStr := field.Type
		if field.IsPointer {
			typeStr = "*" + typeStr
		}

		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = toJSONCamelCase(field.Name)
		}

		result.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, typeStr, jsonTag))
	}
	return result.String()
}

func generateCreateDTOFields(fields []StructField) string {
	var result strings.Builder
	for _, field := range fields {
		dbTag := strings.ToLower(field.DBTag)
		if dbTag == FieldID || dbTag == FieldCreatedAt || dbTag == FieldUpdatedAt {
			continue
		}

		if field.DTOTag == "-" || field.DTOTag == "read" {
			continue
		}

		typeStr := field.Type
		if field.IsPointer {
			typeStr = "*" + typeStr
		}

		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = toJSONCamelCase(field.Name)
		}

		result.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, typeStr, jsonTag))
	}
	return result.String()
}

func generateUpdateDTOFields(fields []StructField) string {
	var result strings.Builder
	for _, field := range fields {
		dbTag := strings.ToLower(field.DBTag)
		if dbTag == FieldID || dbTag == FieldCreatedAt || dbTag == FieldUpdatedAt {
			continue
		}

		if field.DTOTag == "-" || field.DTOTag == "read" {
			continue
		}

		typeStr := field.Type
		if field.IsPointer {
			typeStr = "*" + typeStr
		}

		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = toJSONCamelCase(field.Name)
		}

		result.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, typeStr, jsonTag))
	}
	return result.String()
}

func LoadResourceDTOs() map[string]ResourceDTOs {
	cfg, _ := LoadConfig()
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	dtosDir := cfg.Codegen.Output.DTOs
	if !filepath.IsAbs(dtosDir) {
		dtosDir = filepath.Join(projectRoot, dtosDir)
	}
	if _, err := os.Stat(dtosDir); os.IsNotExist(err) {
		log.Fatal("❌ DTOs directory not found. Run 'make resourcegen' first.")
	}

	files, err := os.ReadDir(dtosDir)
	if err != nil {
		log.Fatalf("❌ Failed to read dtos directory: %v", err)
	}

	resources := make(map[string]ResourceDTOs)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(dtosDir, file.Name())
		resourceName := strings.TrimSuffix(file.Name(), ".go")

		dtos := extractDTOsFromResourceFile(filePath)
		if len(dtos) > 0 {
			resources[resourceName] = ResourceDTOs{
				Name:       resourceName,
				PluralName: Pluralize(resourceName),
				DTOs:       dtos,
			}
		}
	}

	return resources
}

func (r *ResourceDTOs) GetMainDTO() *DTOSchema {
	for name, dto := range r.DTOs {
		if !strings.Contains(name, "Create") && !strings.Contains(name, "Update") {
			return &dto
		}
	}
	return nil
}
