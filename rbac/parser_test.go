package rbac

import (
	"reflect"
	"testing"
)

type TestResource struct {
	ID          string `rbac:"read:*;write:admin"`
	Name        string `rbac:"read:user,editor,admin;write:editor,admin"`
	Email       string `rbac:"read:any;write:admin"`
	Secret      string `rbac:"read:none;write:admin"`
	PublicField string `rbac:"read:*;write:*"`
	NoTag       string
}

type InvalidResource struct {
	Field1 string `rbac:"invalid"`
	Field2 string `rbac:"read:admin"`
}

type InvalidOperationResource struct {
	Field string `rbac:"execute:admin"`
}

type EmptyRolesResource struct {
	Field string `rbac:"read:;write:"`
}

type MixedFormatResource struct {
	Field string `rbac:"read: user , editor ; write: admin "`
}

func TestParseAnnotations(t *testing.T) {
	ClearCache()

	tests := []struct {
		name          string
		resource      interface{}
		expectError   bool
		expectedPerms map[string]Permission
	}{
		{
			name:        "valid resource with multiple permissions",
			resource:    TestResource{},
			expectError: false,
			expectedPerms: map[string]Permission{
				"ID": {
					Field: "ID",
					Read:  []string{"*"},
					Write: []string{"admin"},
				},
				"Name": {
					Field: "Name",
					Read:  []string{"user", "editor", "admin"},
					Write: []string{"editor", "admin"},
				},
				"Email": {
					Field: "Email",
					Read:  []string{"any"},
					Write: []string{"admin"},
				},
				"Secret": {
					Field: "Secret",
					Read:  []string{"none"},
					Write: []string{"admin"},
				},
				"PublicField": {
					Field: "PublicField",
					Read:  []string{"*"},
					Write: []string{"*"},
				},
			},
		},
		{
			name:        "resource with pointer",
			resource:    &TestResource{},
			expectError: false,
			expectedPerms: map[string]Permission{
				"ID": {
					Field: "ID",
					Read:  []string{"*"},
					Write: []string{"admin"},
				},
			},
		},
		{
			name:        "invalid format - missing operation",
			resource:    InvalidResource{},
			expectError: true,
		},
		{
			name:        "invalid operation name",
			resource:    InvalidOperationResource{},
			expectError: true,
		},
		{
			name:          "empty roles are valid",
			resource:      EmptyRolesResource{},
			expectError:   false,
			expectedPerms: map[string]Permission{},
		},
		{
			name:        "mixed format with spaces",
			resource:    MixedFormatResource{},
			expectError: false,
			expectedPerms: map[string]Permission{
				"Field": {
					Field: "Field",
					Read:  []string{"user", "editor"},
					Write: []string{"admin"},
				},
			},
		},
		{
			name:        "non-struct type",
			resource:    "not a struct",
			expectError: true,
		},
		{
			name:        "slice type",
			resource:    []string{"test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms, err := ParseAnnotations(tt.resource)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for fieldName, expectedPerm := range tt.expectedPerms {
				actualPerm, ok := perms[fieldName]
				if !ok {
					t.Errorf("expected permission for field %s, not found", fieldName)
					continue
				}

				if actualPerm.Field != expectedPerm.Field {
					t.Errorf("field %s: expected Field %s, got %s", fieldName, expectedPerm.Field, actualPerm.Field)
				}

				if !reflect.DeepEqual(actualPerm.Read, expectedPerm.Read) {
					t.Errorf("field %s: expected Read %v, got %v", fieldName, expectedPerm.Read, actualPerm.Read)
				}

				if !reflect.DeepEqual(actualPerm.Write, expectedPerm.Write) {
					t.Errorf("field %s: expected Write %v, got %v", fieldName, expectedPerm.Write, actualPerm.Write)
				}
			}
		})
	}
}

func TestParseAnnotationsCaching(t *testing.T) {
	ClearCache()

	resource := TestResource{}

	perms1, err := ParseAnnotations(resource)
	if err != nil {
		t.Fatalf("unexpected error on first parse: %v", err)
	}

	perms2, err := ParseAnnotations(resource)
	if err != nil {
		t.Fatalf("unexpected error on second parse: %v", err)
	}

	if !reflect.DeepEqual(perms1, perms2) {
		t.Error("cached permissions don't match original")
	}

	ptrResource := &TestResource{}
	perms3, err := ParseAnnotations(ptrResource)
	if err != nil {
		t.Fatalf("unexpected error on pointer parse: %v", err)
	}

	if !reflect.DeepEqual(perms1, perms3) {
		t.Error("pointer and value type should produce same cached result")
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		tag         string
		expectError bool
		expected    Permission
	}{
		{
			name:        "simple read permission",
			fieldName:   "Field",
			tag:         "read:admin",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"admin"},
				Write: []string{},
			},
		},
		{
			name:        "simple write permission",
			fieldName:   "Field",
			tag:         "write:editor",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{},
				Write: []string{"editor"},
			},
		},
		{
			name:        "read and write permissions",
			fieldName:   "Field",
			tag:         "read:user,editor;write:admin",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"user", "editor"},
				Write: []string{"admin"},
			},
		},
		{
			name:        "special keyword - star",
			fieldName:   "Field",
			tag:         "read:*;write:*",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"*"},
				Write: []string{"*"},
			},
		},
		{
			name:        "special keyword - any",
			fieldName:   "Field",
			tag:         "read:any;write:any",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"any"},
				Write: []string{"any"},
			},
		},
		{
			name:        "special keyword - none",
			fieldName:   "Field",
			tag:         "read:none;write:none",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"none"},
				Write: []string{"none"},
			},
		},
		{
			name:        "invalid format - no colon",
			fieldName:   "Field",
			tag:         "admin",
			expectError: true,
		},
		{
			name:        "empty roles after colon",
			fieldName:   "Field",
			tag:         "read:;write:",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  nil,
				Write: nil,
			},
		},
		{
			name:        "invalid operation",
			fieldName:   "Field",
			tag:         "delete:admin",
			expectError: true,
		},
		{
			name:        "case insensitive operations",
			fieldName:   "Field",
			tag:         "READ:admin;WRITE:editor",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"admin"},
				Write: []string{"editor"},
			},
		},
		{
			name:        "whitespace handling",
			fieldName:   "Field",
			tag:         " read : user , editor ; write : admin ",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"user", "editor"},
				Write: []string{"admin"},
			},
		},
		{
			name:        "multiple roles with spaces",
			fieldName:   "Field",
			tag:         "read: role1 , role2 , role3",
			expectError: false,
			expected: Permission{
				Field: "Field",
				Read:  []string{"role1", "role2", "role3"},
				Write: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm, err := parseTag(tt.fieldName, tt.tag)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if perm.Field != tt.expected.Field {
				t.Errorf("expected Field %s, got %s", tt.expected.Field, perm.Field)
			}

			if !reflect.DeepEqual(perm.Read, tt.expected.Read) {
				t.Errorf("expected Read %v, got %v", tt.expected.Read, perm.Read)
			}

			if !reflect.DeepEqual(perm.Write, tt.expected.Write) {
				t.Errorf("expected Write %v, got %v", tt.expected.Write, perm.Write)
			}
		})
	}
}

func TestHasAnnotation(t *testing.T) {
	resource := TestResource{}

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{
			name:      "field with annotation",
			fieldName: "ID",
			expected:  true,
		},
		{
			name:      "field without annotation",
			fieldName: "NoTag",
			expected:  false,
		},
		{
			name:      "non-existent field",
			fieldName: "DoesNotExist",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAnnotation(resource, tt.fieldName)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}

	ptrResource := &TestResource{}
	result := HasAnnotation(ptrResource, "ID")
	if !result {
		t.Error("HasAnnotation should work with pointer types")
	}

	nonStruct := "not a struct"
	result = HasAnnotation(nonStruct, "Field")
	if result {
		t.Error("HasAnnotation should return false for non-struct types")
	}
}

func TestGetFieldNames(t *testing.T) {
	resource := TestResource{}

	fields := GetFieldNames(resource)

	expectedFields := []string{"ID", "Name", "Email", "Secret", "PublicField", "NoTag"}

	if len(fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(fields))
	}

	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[field] = true
	}

	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("expected field %s not found in result", expected)
		}
	}

	ptrResource := &TestResource{}
	fields = GetFieldNames(ptrResource)
	if len(fields) != len(expectedFields) {
		t.Error("GetFieldNames should work with pointer types")
	}

	nonStruct := "not a struct"
	fields = GetFieldNames(nonStruct)
	if fields != nil {
		t.Error("GetFieldNames should return nil for non-struct types")
	}
}

func TestClearCache(t *testing.T) {
	ClearCache()

	resource := TestResource{}

	_, err := ParseAnnotations(resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	annotationCacheMu.RLock()
	cacheSize := len(annotationCache)
	annotationCacheMu.RUnlock()

	if cacheSize == 0 {
		t.Error("expected cache to be populated")
	}

	ClearCache()

	annotationCacheMu.RLock()
	cacheSize = len(annotationCache)
	annotationCacheMu.RUnlock()

	if cacheSize != 0 {
		t.Error("expected cache to be empty after clear")
	}
}

type UnexportedFieldsResource struct {
	Exported   string `rbac:"read:admin"`
	unexported string //lint:ignore U1000 intentionally unused field for testing unexported field handling
}

func TestParseAnnotationsUnexportedFields(t *testing.T) {
	ClearCache()

	resource := UnexportedFieldsResource{}
	perms, err := ParseAnnotations(resource)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := perms["Exported"]; !ok {
		t.Error("expected exported field to be in permissions")
	}

	if _, ok := perms["unexported"]; ok {
		t.Error("unexported field should not be in permissions")
	}
}
