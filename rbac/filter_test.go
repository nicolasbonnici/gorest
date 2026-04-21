package rbac

import (
	"reflect"
	"testing"
)

type FilterTestResource struct {
	PublicField string `rbac:"read:*;write:*"`
	UserField   string `rbac:"read:user,editor,admin;write:editor,admin"`
	AdminField  string `rbac:"read:admin;write:admin"`
	NoTagField  string
}

func validateDenyPolicyUserRole(t *testing.T, result interface{}) {
	res, ok := result.(FilterTestResource)
	if !ok {
		t.Fatalf("expected FilterTestResource, got %T", result)
	}
	if res.PublicField != "public" {
		t.Error("PublicField should be visible")
	}
	if res.UserField != "user" {
		t.Error("UserField should be visible for user role")
	}
	if res.AdminField != "" {
		t.Error("AdminField should be filtered for user role")
	}
	if res.NoTagField != "" {
		t.Error("NoTagField should be filtered with deny policy")
	}
}

func validateAllowPolicyUserRole(t *testing.T, result interface{}) {
	res, ok := result.(FilterTestResource)
	if !ok {
		t.Fatalf("expected FilterTestResource, got %T", result)
	}
	if res.NoTagField != "notag" {
		t.Error("NoTagField should be visible with allow policy")
	}
}

func validateNoRoles(t *testing.T, result interface{}) {
	res, ok := result.(FilterTestResource)
	if !ok {
		t.Fatalf("expected FilterTestResource, got %T", result)
	}
	if res.PublicField != "public" {
		t.Error("PublicField should be visible")
	}
	if res.UserField != "" {
		t.Error("UserField should be filtered with no roles")
	}
}

func validatePointerResource(t *testing.T, result interface{}) {
	res, ok := result.(*FilterTestResource)
	if !ok {
		t.Fatalf("expected *FilterTestResource, got %T", result)
	}
	if res.PublicField != "public" {
		t.Error("PublicField should be visible")
	}
}

func TestFilterReadFields(t *testing.T) {
	ClearCache()

	tests := []struct {
		name           string
		resource       interface{}
		userRoles      []string
		config         Config
		expectError    bool
		validateResult func(*testing.T, interface{})
	}{
		{
			name: "filter with deny policy - user role",
			resource: FilterTestResource{
				PublicField: "public",
				UserField:   "user",
				AdminField:  "admin",
				NoTagField:  "notag",
			},
			userRoles: []string{"user"},
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				DefaultFieldPolicy: "deny",
			},
			expectError:    false,
			validateResult: validateDenyPolicyUserRole,
		},
		{
			name: "filter with allow policy - user role",
			resource: FilterTestResource{
				PublicField: "public",
				UserField:   "user",
				AdminField:  "admin",
				NoTagField:  "notag",
			},
			userRoles: []string{"user"},
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				DefaultFieldPolicy: "allow",
			},
			expectError:    false,
			validateResult: validateAllowPolicyUserRole,
		},
		{
			name: "filter with no roles",
			resource: FilterTestResource{
				PublicField: "public",
				UserField:   "user",
				AdminField:  "admin",
				NoTagField:  "notag",
			},
			userRoles: []string{},
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				DefaultFieldPolicy: "deny",
			},
			expectError:    false,
			validateResult: validateNoRoles,
		},
		{
			name: "filter pointer resource",
			resource: &FilterTestResource{
				PublicField: "public",
				UserField:   "user",
				AdminField:  "admin",
			},
			userRoles: []string{"user"},
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				DefaultFieldPolicy: "deny",
			},
			expectError:    false,
			validateResult: validatePointerResource,
		},
		{
			name:        "invalid resource type",
			resource:    "not a struct",
			userRoles:   []string{"user"},
			config:      DefaultConfig(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runFilterReadFieldsTest(t, tt.resource, tt.userRoles, tt.config, tt.expectError, tt.validateResult)
		})
	}
}

func runFilterReadFieldsTest(t *testing.T, resource interface{}, userRoles []string, config Config, expectError bool, validate func(*testing.T, interface{})) {
	result, err := filterReadFields(resource, userRoles, config)

	if expectError {
		if err == nil {
			t.Error("expected error, got nil")
		}
		return
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if validate != nil {
		validate(t, result)
	}
}

func TestFilterSlice(t *testing.T) {
	ClearCache()

	config := Config{
		DefaultPolicy:      DenyAll,
		SuperuserRole:      "admin",
		RoleHierarchy:      make(map[string][]string),
		DefaultFieldPolicy: "deny",
	}

	tests := []struct {
		name        string
		resource    interface{}
		userRoles   []string
		expectError bool
		validate    func(*testing.T, interface{})
	}{
		{
			name: "filter slice of structs",
			resource: []FilterTestResource{
				{PublicField: "public1", UserField: "user1", AdminField: "admin1"},
				{PublicField: "public2", UserField: "user2", AdminField: "admin2"},
			},
			userRoles:   []string{"user"},
			expectError: false,
			validate: func(t *testing.T, result interface{}) {
				res, ok := result.([]FilterTestResource)
				if !ok {
					t.Fatalf("expected []FilterTestResource, got %T", result)
				}

				if len(res) != 2 {
					t.Fatalf("expected 2 items, got %d", len(res))
				}

				if res[0].PublicField != "public1" {
					t.Error("first item PublicField should be visible")
				}
				if res[0].AdminField != "" {
					t.Error("first item AdminField should be filtered")
				}
			},
		},
		{
			name: "filter slice of pointers",
			resource: []*FilterTestResource{
				{PublicField: "public1", UserField: "user1"},
				{PublicField: "public2", UserField: "user2"},
			},
			userRoles:   []string{"user"},
			expectError: false,
			validate: func(t *testing.T, result interface{}) {
				res, ok := result.([]*FilterTestResource)
				if !ok {
					t.Fatalf("expected []*FilterTestResource, got %T", result)
				}

				if len(res) != 2 {
					t.Fatalf("expected 2 items, got %d", len(res))
				}

				if res[0].PublicField != "public1" {
					t.Error("first item PublicField should be visible")
				}
			},
		},
		{
			name:        "non-slice type",
			resource:    FilterTestResource{},
			userRoles:   []string{"user"},
			expectError: true,
		},
		{
			name:        "empty slice",
			resource:    []FilterTestResource{},
			userRoles:   []string{"user"},
			expectError: false,
			validate: func(t *testing.T, result interface{}) {
				res, ok := result.([]FilterTestResource)
				if !ok {
					t.Fatalf("expected []FilterTestResource, got %T", result)
				}

				if len(res) != 0 {
					t.Errorf("expected 0 items, got %d", len(res))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterSlice(tt.resource, tt.userRoles, config)

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

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCanReadField(t *testing.T) {
	permissions := PermissionSet{
		"PublicField": {
			Field: "PublicField",
			Read:  []string{"*"},
		},
		"AdminField": {
			Field: "AdminField",
			Read:  []string{"admin"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canReadField(tt.fieldName, permissions, tt.userRoles, tt.config)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFilterReadFieldsPreservesTypes(t *testing.T) {
	ClearCache()

	type ComplexResource struct {
		IntField    int               `rbac:"read:*"`
		FloatField  float64           `rbac:"read:admin"`
		BoolField   bool              `rbac:"read:*"`
		StringSlice []string          `rbac:"read:admin"`
		MapField    map[string]string `rbac:"read:*"`
	}

	resource := ComplexResource{
		IntField:    42,
		FloatField:  3.14,
		BoolField:   true,
		StringSlice: []string{"a", "b"},
		MapField:    map[string]string{"key": "value"},
	}

	config := Config{
		DefaultPolicy:      DenyAll,
		SuperuserRole:      "admin",
		RoleHierarchy:      make(map[string][]string),
		DefaultFieldPolicy: "deny",
	}

	result, err := filterReadFields(resource, []string{"user"}, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res, ok := result.(ComplexResource)
	if !ok {
		t.Fatalf("expected ComplexResource, got %T", result)
	}

	if res.IntField != 42 {
		t.Errorf("IntField should be preserved, got %d", res.IntField)
	}

	if res.FloatField != 0 {
		t.Errorf("FloatField should be zero, got %f", res.FloatField)
	}

	if res.BoolField != true {
		t.Errorf("BoolField should be preserved, got %v", res.BoolField)
	}

	if res.StringSlice != nil {
		t.Errorf("StringSlice should be nil, got %v", res.StringSlice)
	}

	if !reflect.DeepEqual(res.MapField, map[string]string{"key": "value"}) {
		t.Errorf("MapField should be preserved")
	}
}
