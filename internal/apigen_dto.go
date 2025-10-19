package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DTOSchema represents a DTO structure
type DTOSchema struct {
	Name   string
	Fields []StructField
}

// ResourceDTOs represents DTOs for a resource
type ResourceDTOs struct {
	Name       string
	PluralName string
	DTOs       map[string]DTOSchema
}

// generateDTOForStruct generates DTO file for a struct
func generateDTOForStruct(dtosDir string, structName string) {
	dtoFile := filepath.Join(dtosDir, strings.ToLower(structName)+".go")

	projectRoot, _ := findProjectRoot()
	modelPath := filepath.Join(projectRoot, "internal", "api", "models", strings.ToLower(structName)+".go")
	fields := extractStructFields(modelPath, structName)

	code := generateDTOsFromModel(structName, fields)
	if err := os.WriteFile(dtoFile, []byte(code), 0644); err != nil {
		log.Fatalf("failed to write DTOs for %s: %v", structName, err)
	}
	log.Printf("📝 Generated DTOs for model: %s → %s", structName, dtoFile)
}

// generateDTOsFromModel generates DTO code for a model
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

// generateDTOFields generates fields for response DTO (excludes sensitive fields)
func generateDTOFields(fields []StructField) string {
	var result strings.Builder
	for _, field := range fields {
		// Skip sensitive fields in Response DTO
		if isSensitiveField(field.Name) {
			continue
		}

		// Build type string
		typeStr := field.Type
		if field.IsPointer {
			typeStr = "*" + typeStr
		}

		// Build JSON tag from existing json tag
		jsonTag := field.JSONTag
		if jsonTag == "" {
			// If no json tag exists, use field name in lowercase
			jsonTag = strings.ToLower(field.Name)
		}

		result.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, typeStr, jsonTag))
	}
	return result.String()
}

// generateCreateDTOFields generates fields for Create DTO (excludes auto-generated fields)
func generateCreateDTOFields(fields []StructField) string {
	var result strings.Builder
	for _, field := range fields {
		// Exclude auto-generated fields from Create DTO
		dbTag := strings.ToLower(field.DBTag)
		if dbTag == FieldID || dbTag == FieldCreatedAt || dbTag == FieldUpdatedAt {
			continue
		}

		typeStr := field.Type
		if field.IsPointer {
			typeStr = "*" + typeStr
		}

		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = strings.ToLower(field.Name)
		}

		result.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, typeStr, jsonTag))
	}
	return result.String()
}

// generateUpdateDTOFields generates fields for Update DTO (excludes auto-generated fields)
func generateUpdateDTOFields(fields []StructField) string {
	var result strings.Builder
	for _, field := range fields {
		// Exclude auto-generated fields from Update DTO
		dbTag := strings.ToLower(field.DBTag)
		if dbTag == FieldID || dbTag == FieldCreatedAt || dbTag == FieldUpdatedAt {
			continue
		}

		typeStr := field.Type
		if field.IsPointer {
			typeStr = "*" + typeStr
		}

		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = strings.ToLower(field.Name)
		}

		result.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", field.Name, typeStr, jsonTag))
	}
	return result.String()
}

// LoadResourceDTOs loads all DTOs from the dtos directory
func LoadResourceDTOs() map[string]ResourceDTOs {
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	dtosDir := filepath.Join(projectRoot, "internal", "api", "dtos")
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
				PluralName: pluralize(resourceName),
				DTOs:       dtos,
			}
		}
	}

	return resources
}

// GetMainDTO returns the main DTO (not Create or Update)
func (r *ResourceDTOs) GetMainDTO() *DTOSchema {
	for name, dto := range r.DTOs {
		if !strings.Contains(name, "Create") && !strings.Contains(name, "Update") {
			return &dto
		}
	}
	return nil
}
