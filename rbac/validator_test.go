package rbac

import (
	"reflect"
	"testing"
)

type ValidatorTestResource struct {
	PublicField string `rbac:"read:*;write:*"`
	UserField   string `rbac:"read:user;write:editor,admin"`
	AdminField  string `rbac:"read:admin;write:admin"`
	NoTagField  string
}

func TestValidateWriteFields(t *testing.T) {
	ClearCache()

	tests := []struct {
		name            string
		resource        interface{}
		userRoles       []string
		config          Config
		expectError     bool
		forbiddenFields []string
	}{
		{
			name: "all fields allowed",
			resource: ValidatorTestResource{
				PublicField: "value",
			},
			userRoles:   []string{"user"},
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "forbidden field - admin field with user role",
			resource: ValidatorTestResource{
				AdminField: "value",
			},
			userRoles:       []string{"user"},
			config:          DefaultConfig(),
			expectError:     true,
			forbiddenFields: []string{"AdminField"},
		},
		{
			name: "multiple forbidden fields",
			resource: ValidatorTestResource{
				UserField:  "value1",
				AdminField: "value2",
			},
			userRoles:       []string{"user"},
			config:          DefaultConfig(),
			expectError:     true,
			forbiddenFields: []string{"UserField", "AdminField"},
		},
		{
			name: "zero values are ignored",
			resource: ValidatorTestResource{
				AdminField: "",
			},
			userRoles:   []string{"user"},
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "editor can write user field",
			resource: ValidatorTestResource{
				UserField: "value",
			},
			userRoles:   []string{"editor"},
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "no tag field with deny policy",
			resource: ValidatorTestResource{
				NoTagField: "value",
			},
			userRoles: []string{"user"},
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				DefaultFieldPolicy: "deny",
			},
			expectError:     true,
			forbiddenFields: []string{"NoTagField"},
		},
		{
			name: "no tag field with allow policy",
			resource: ValidatorTestResource{
				NoTagField: "value",
			},
			userRoles: []string{"user"},
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				DefaultFieldPolicy: "allow",
			},
			expectError: false,
		},
		{
			name: "pointer resource",
			resource: &ValidatorTestResource{
				AdminField: "value",
			},
			userRoles:       []string{"user"},
			config:          DefaultConfig(),
			expectError:     true,
			forbiddenFields: []string{"AdminField"},
		},
		{
			name:        "non-struct resource",
			resource:    "not a struct",
			userRoles:   []string{"user"},
			config:      DefaultConfig(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWriteFields(tt.resource, tt.userRoles, tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				if len(tt.forbiddenFields) > 0 {
					validationErr, ok := err.(*ValidationError)
					if !ok {
						t.Errorf("expected ValidationError, got %T", err)
						return
					}

					if len(validationErr.ForbiddenFields) != len(tt.forbiddenFields) {
						t.Errorf("expected %d forbidden fields, got %d", len(tt.forbiddenFields), len(validationErr.ForbiddenFields))
					}

					for _, expected := range tt.forbiddenFields {
						found := false
						for _, actual := range validationErr.ForbiddenFields {
							if actual == expected {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected forbidden field %s not found", expected)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCanWriteField(t *testing.T) {
	permissions := PermissionSet{
		"PublicField": {
			Field: "PublicField",
			Write: []string{"*"},
		},
		"AdminField": {
			Field: "AdminField",
			Write: []string{"admin"},
		},
	}

	tests := []struct {
		name      string
		fieldName string
		userRoles []string
		config    Config
		expected  bool
	}{
		{
			name:      "field with permission - allowed",
			fieldName: "PublicField",
			userRoles: []string{"user"},
			config:    DefaultConfig(),
			expected:  true,
		},
		{
			name:      "field with permission - denied",
			fieldName: "AdminField",
			userRoles: []string{"user"},
			config:    DefaultConfig(),
			expected:  false,
		},
		{
			name:      "field without permission - allow policy",
			fieldName: "NoTagField",
			userRoles: []string{"user"},
			config: Config{
				DefaultFieldPolicy: "allow",
			},
			expected: true,
		},
		{
			name:      "field without permission - deny policy",
			fieldName: "NoTagField",
			userRoles: []string{"user"},
			config: Config{
				DefaultFieldPolicy: "deny",
			},
			expected: false,
		},
		{
			name:      "admin role can write admin field",
			fieldName: "AdminField",
			userRoles: []string{"admin"},
			config:    DefaultConfig(),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canWriteField(tt.fieldName, permissions, tt.userRoles, tt.config)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{
			name:     "zero string",
			value:    "",
			expected: true,
		},
		{
			name:     "non-zero string",
			value:    "hello",
			expected: false,
		},
		{
			name:     "zero int",
			value:    0,
			expected: true,
		},
		{
			name:     "non-zero int",
			value:    42,
			expected: false,
		},
		{
			name:     "zero float",
			value:    0.0,
			expected: true,
		},
		{
			name:     "non-zero float",
			value:    3.14,
			expected: false,
		},
		{
			name:     "false bool",
			value:    false,
			expected: true,
		},
		{
			name:     "true bool",
			value:    true,
			expected: false,
		},
		{
			name:     "nil pointer",
			value:    (*string)(nil),
			expected: true,
		},
		{
			name:     "nil slice",
			value:    []string(nil),
			expected: true,
		},
		{
			name:     "empty slice",
			value:    []string{},
			expected: true,
		},
		{
			name:     "non-empty slice",
			value:    []string{"a"},
			expected: false,
		},
		{
			name:     "nil map",
			value:    map[string]string(nil),
			expected: true,
		},
		{
			name:     "empty map",
			value:    map[string]string{},
			expected: true,
		},
		{
			name:     "non-empty map",
			value:    map[string]string{"key": "value"},
			expected: false,
		},
		{
			name:     "empty array",
			value:    [0]int{},
			expected: true,
		},
		{
			name:     "zero uint",
			value:    uint(0),
			expected: true,
		},
		{
			name:     "non-zero uint",
			value:    uint(42),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isZeroValue(v)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsStructZero(t *testing.T) {
	type TestStruct struct {
		Field1 string
		Field2 int
		Field3 bool
	}

	tests := []struct {
		name     string
		value    TestStruct
		expected bool
	}{
		{
			name:     "zero struct",
			value:    TestStruct{},
			expected: true,
		},
		{
			name: "non-zero struct - one field",
			value: TestStruct{
				Field1: "value",
			},
			expected: false,
		},
		{
			name: "non-zero struct - multiple fields",
			value: TestStruct{
				Field1: "value",
				Field2: 42,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isStructZero(v)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateWriteFieldsComplexTypes(t *testing.T) {
	ClearCache()

	type ComplexResource struct {
		IntField    int               `rbac:"write:admin"`
		FloatField  float64           `rbac:"write:admin"`
		BoolField   bool              `rbac:"write:admin"`
		StringSlice []string          `rbac:"write:admin"`
		MapField    map[string]string `rbac:"write:admin"`
		PtrField    *string           `rbac:"write:admin"`
	}

	config := DefaultConfig()

	str := "value"

	tests := []struct {
		name            string
		resource        ComplexResource
		userRoles       []string
		expectError     bool
		forbiddenFields []string
	}{
		{
			name:        "all zero values",
			resource:    ComplexResource{},
			userRoles:   []string{"user"},
			expectError: false,
		},
		{
			name: "non-zero int",
			resource: ComplexResource{
				IntField: 42,
			},
			userRoles:       []string{"user"},
			expectError:     true,
			forbiddenFields: []string{"IntField"},
		},
		{
			name: "non-zero float",
			resource: ComplexResource{
				FloatField: 3.14,
			},
			userRoles:       []string{"user"},
			expectError:     true,
			forbiddenFields: []string{"FloatField"},
		},
		{
			name: "true bool",
			resource: ComplexResource{
				BoolField: true,
			},
			userRoles:       []string{"user"},
			expectError:     true,
			forbiddenFields: []string{"BoolField"},
		},
		{
			name: "non-empty slice",
			resource: ComplexResource{
				StringSlice: []string{"a", "b"},
			},
			userRoles:       []string{"user"},
			expectError:     true,
			forbiddenFields: []string{"StringSlice"},
		},
		{
			name: "non-empty map",
			resource: ComplexResource{
				MapField: map[string]string{"key": "value"},
			},
			userRoles:       []string{"user"},
			expectError:     true,
			forbiddenFields: []string{"MapField"},
		},
		{
			name: "non-nil pointer",
			resource: ComplexResource{
				PtrField: &str,
			},
			userRoles:       []string{"user"},
			expectError:     true,
			forbiddenFields: []string{"PtrField"},
		},
		{
			name: "admin can write all",
			resource: ComplexResource{
				IntField:    42,
				FloatField:  3.14,
				BoolField:   true,
				StringSlice: []string{"a"},
				MapField:    map[string]string{"key": "value"},
				PtrField:    &str,
			},
			userRoles:   []string{"admin"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWriteFields(tt.resource, tt.userRoles, config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected ValidationError, got %T", err)
					return
				}

				if len(validationErr.ForbiddenFields) != len(tt.forbiddenFields) {
					t.Errorf("expected %d forbidden fields, got %d", len(tt.forbiddenFields), len(validationErr.ForbiddenFields))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsZeroValueInvalidValue(t *testing.T) {
	var v reflect.Value
	result := isZeroValue(v)
	if !result {
		t.Error("invalid reflect.Value should be considered zero")
	}
}

func TestValidateWriteFieldsNestedStruct(t *testing.T) {
	ClearCache()

	type NestedStruct struct {
		Field1 string
		Field2 int
	}

	type ParentResource struct {
		Nested NestedStruct `rbac:"write:admin"`
	}

	config := DefaultConfig()

	tests := []struct {
		name        string
		resource    ParentResource
		userRoles   []string
		expectError bool
	}{
		{
			name:        "zero nested struct",
			resource:    ParentResource{},
			userRoles:   []string{"user"},
			expectError: false,
		},
		{
			name: "non-zero nested struct",
			resource: ParentResource{
				Nested: NestedStruct{
					Field1: "value",
					Field2: 42,
				},
			},
			userRoles:   []string{"user"},
			expectError: true,
		},
		{
			name: "admin can write nested struct",
			resource: ParentResource{
				Nested: NestedStruct{
					Field1: "value",
					Field2: 42,
				},
			},
			userRoles:   []string{"admin"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWriteFields(tt.resource, tt.userRoles, config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestStrictValidation(t *testing.T) {
	ClearCache()

	type StrictTestResource struct {
		PublicField       string `rbac:"read:*;write:*"`
		AdminField        string `rbac:"read:admin;write:admin"`
		IntField          int    `rbac:"write:admin"`
		BoolField         bool   `rbac:"write:admin"`
		NoTagField        string
		AnotherNoTagField int
	}

	tests := []struct {
		name            string
		resource        StrictTestResource
		userRoles       []string
		strictMode      bool
		expectError     bool
		forbiddenFields []string
	}{
		{
			name: "strict mode disabled - zero values bypass validation",
			resource: StrictTestResource{
				AdminField: "",
				IntField:   0,
				BoolField:  false,
			},
			userRoles:   []string{"user"},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "strict mode enabled - zero string validated",
			resource: StrictTestResource{
				PublicField: "allowed",
				AdminField:  "",
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"AdminField", "IntField", "BoolField", "NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode enabled - zero int validated",
			resource: StrictTestResource{
				PublicField: "allowed",
				IntField:    0,
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"AdminField", "IntField", "BoolField", "NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode enabled - false bool validated",
			resource: StrictTestResource{
				PublicField: "allowed",
				BoolField:   false,
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"AdminField", "IntField", "BoolField", "NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode enabled - all zero fields validated",
			resource: StrictTestResource{
				PublicField: "allowed",
				AdminField:  "",
				IntField:    0,
				BoolField:   false,
				NoTagField:  "",
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"AdminField", "IntField", "BoolField", "NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode enabled - admin blocked by zero-valued NoTagField with deny policy",
			resource: StrictTestResource{
				PublicField: "allowed",
				AdminField:  "",
				IntField:    0,
				BoolField:   false,
			},
			userRoles:       []string{"admin"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode enabled - public field allowed with zero value",
			resource: StrictTestResource{
				PublicField: "",
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"AdminField", "IntField", "BoolField", "NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode enabled - no tag field with deny policy",
			resource: StrictTestResource{
				PublicField: "allowed",
				NoTagField:  "",
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"AdminField", "IntField", "BoolField", "NoTagField", "AnotherNoTagField"},
		},
		{
			name: "strict mode disabled - non-zero values still validated",
			resource: StrictTestResource{
				AdminField: "value",
			},
			userRoles:       []string{"user"},
			strictMode:      false,
			expectError:     true,
			forbiddenFields: []string{"AdminField"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.StrictValidation = tt.strictMode

			err := validateWriteFields(tt.resource, tt.userRoles, config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				if len(tt.forbiddenFields) > 0 {
					validationErr, ok := err.(*ValidationError)
					if !ok {
						t.Errorf("expected ValidationError, got %T", err)
						return
					}

					if len(validationErr.ForbiddenFields) != len(tt.forbiddenFields) {
						t.Errorf("expected %d forbidden fields, got %d", len(tt.forbiddenFields), len(validationErr.ForbiddenFields))
					}

					for _, expected := range tt.forbiddenFields {
						found := false
						for _, actual := range validationErr.ForbiddenFields {
							if actual == expected {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected forbidden field %s not found in %v", expected, validationErr.ForbiddenFields)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestStrictValidationComplexTypes(t *testing.T) {
	ClearCache()

	type ComplexStrictResource struct {
		PublicField string            `rbac:"write:*"`
		SliceField  []string          `rbac:"write:admin"`
		MapField    map[string]string `rbac:"write:admin"`
		PtrField    *string           `rbac:"write:admin"`
	}

	tests := []struct {
		name            string
		resource        ComplexStrictResource
		userRoles       []string
		strictMode      bool
		expectError     bool
		forbiddenFields []string
	}{
		{
			name: "strict mode disabled - nil slice bypasses validation",
			resource: ComplexStrictResource{
				SliceField: nil,
			},
			userRoles:   []string{"user"},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "strict mode enabled - nil slice validated",
			resource: ComplexStrictResource{
				PublicField: "allowed",
				SliceField:  nil,
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"SliceField", "MapField", "PtrField"},
		},
		{
			name: "strict mode disabled - empty slice bypasses validation",
			resource: ComplexStrictResource{
				SliceField: []string{},
			},
			userRoles:   []string{"user"},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "strict mode enabled - empty slice validated",
			resource: ComplexStrictResource{
				PublicField: "allowed",
				SliceField:  []string{},
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"SliceField", "MapField", "PtrField"},
		},
		{
			name: "strict mode disabled - nil map bypasses validation",
			resource: ComplexStrictResource{
				MapField: nil,
			},
			userRoles:   []string{"user"},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "strict mode enabled - nil map validated",
			resource: ComplexStrictResource{
				PublicField: "allowed",
				MapField:    nil,
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"SliceField", "MapField", "PtrField"},
		},
		{
			name: "strict mode disabled - nil pointer bypasses validation",
			resource: ComplexStrictResource{
				PtrField: nil,
			},
			userRoles:   []string{"user"},
			strictMode:  false,
			expectError: false,
		},
		{
			name: "strict mode enabled - nil pointer validated",
			resource: ComplexStrictResource{
				PublicField: "allowed",
				PtrField:    nil,
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"SliceField", "MapField", "PtrField"},
		},
		{
			name: "strict mode enabled - all nil complex types validated",
			resource: ComplexStrictResource{
				PublicField: "allowed",
				SliceField:  nil,
				MapField:    nil,
				PtrField:    nil,
			},
			userRoles:       []string{"user"},
			strictMode:      true,
			expectError:     true,
			forbiddenFields: []string{"SliceField", "MapField", "PtrField"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.StrictValidation = tt.strictMode

			err := validateWriteFields(tt.resource, tt.userRoles, config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				if len(tt.forbiddenFields) > 0 {
					validationErr, ok := err.(*ValidationError)
					if !ok {
						t.Errorf("expected ValidationError, got %T", err)
						return
					}

					if len(validationErr.ForbiddenFields) != len(tt.forbiddenFields) {
						t.Errorf("expected %d forbidden fields, got %d", len(tt.forbiddenFields), len(validationErr.ForbiddenFields))
					}

					for _, expected := range tt.forbiddenFields {
						found := false
						for _, actual := range validationErr.ForbiddenFields {
							if actual == expected {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected forbidden field %s not found in %v", expected, validationErr.ForbiddenFields)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
