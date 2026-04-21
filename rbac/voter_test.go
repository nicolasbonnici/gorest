package rbac

import (
	"context"
	"strings"
	"testing"
)

type VoterTestResource struct {
	PublicField string `rbac:"read:*;write:*"`
	UserField   string `rbac:"read:user,editor,admin;write:editor,admin"`
	AdminField  string `rbac:"read:admin;write:admin"`
	AnyField    string `rbac:"read:any;write:any"`
	NoneField   string `rbac:"read:none;write:none"`
	NoTagField  string
}

func TestNewVoter(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name:        "valid config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "invalid config - empty superuser role",
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				DefaultFieldPolicy: "deny",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			voter, err := NewVoter(tt.config)

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

			if voter == nil {
				t.Error("expected non-nil voter")
			}
		})
	}
}

func TestCheckRead(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	config.SuperuserRole = "superadmin"
	config.RoleHierarchy = map[string][]string{
		"admin":  {"editor"},
		"editor": {"user"},
	}

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	tests := []struct {
		name        string
		roles       []string
		field       string
		expectError bool
	}{
		{
			name:        "public field - no roles",
			roles:       []string{},
			field:       "PublicField",
			expectError: false,
		},
		{
			name:        "public field - with roles",
			roles:       []string{"user"},
			field:       "PublicField",
			expectError: false,
		},
		{
			name:        "user field - user role",
			roles:       []string{"user"},
			field:       "UserField",
			expectError: false,
		},
		{
			name:        "user field - editor role",
			roles:       []string{"editor"},
			field:       "UserField",
			expectError: false,
		},
		{
			name:        "user field - admin role (inherited)",
			roles:       []string{"admin"},
			field:       "UserField",
			expectError: false,
		},
		{
			name:        "admin field - user role",
			roles:       []string{"user"},
			field:       "AdminField",
			expectError: true,
		},
		{
			name:        "admin field - admin role",
			roles:       []string{"admin"},
			field:       "AdminField",
			expectError: false,
		},
		{
			name:        "any field - with roles",
			roles:       []string{"user"},
			field:       "AnyField",
			expectError: false,
		},
		{
			name:        "any field - no roles",
			roles:       []string{},
			field:       "AnyField",
			expectError: true,
		},
		{
			name:        "none field - admin role",
			roles:       []string{"admin"},
			field:       "NoneField",
			expectError: true,
		},
		{
			name:        "no tag field - deny policy",
			roles:       []string{"user"},
			field:       "NoTagField",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRoles(context.Background(), tt.roles)
			resource := VoterTestResource{}

			err := voter.CheckRead(ctx, resource, tt.field)

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

func TestCheckReadSuperuser(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	config.SuperuserRole = "admin"

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	ctx := WithRoles(context.Background(), []string{"admin"})
	resource := VoterTestResource{}

	fields := []string{"PublicField", "UserField", "AdminField", "AnyField", "NoneField", "NoTagField"}

	for _, field := range fields {
		err := voter.CheckRead(ctx, resource, field)
		if err != nil {
			t.Errorf("superuser should have access to field %s, got error: %v", field, err)
		}
	}
}

func TestCheckReadValidation(t *testing.T) {
	config := DefaultConfig()
	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	ctx := context.Background()

	err = voter.CheckRead(ctx, nil, "Field")
	if err == nil || !strings.Contains(err.Error(), "resource cannot be nil") {
		t.Error("expected error for nil resource")
	}

	resource := VoterTestResource{}
	err = voter.CheckRead(ctx, resource, "")
	if err == nil || !strings.Contains(err.Error(), "field name cannot be empty") {
		t.Error("expected error for empty field name")
	}
}

func TestCheckWrite(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	config.SuperuserRole = "superadmin"
	config.RoleHierarchy = map[string][]string{
		"admin":  {"editor"},
		"editor": {"user"},
	}

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	tests := []struct {
		name        string
		roles       []string
		field       string
		expectError bool
	}{
		{
			name:        "public field - no roles",
			roles:       []string{},
			field:       "PublicField",
			expectError: false,
		},
		{
			name:        "user field - user role",
			roles:       []string{"user"},
			field:       "UserField",
			expectError: true,
		},
		{
			name:        "user field - editor role",
			roles:       []string{"editor"},
			field:       "UserField",
			expectError: false,
		},
		{
			name:        "user field - admin role (inherited)",
			roles:       []string{"admin"},
			field:       "UserField",
			expectError: false,
		},
		{
			name:        "admin field - editor role",
			roles:       []string{"editor"},
			field:       "AdminField",
			expectError: true,
		},
		{
			name:        "admin field - admin role",
			roles:       []string{"admin"},
			field:       "AdminField",
			expectError: false,
		},
		{
			name:        "any field - with roles",
			roles:       []string{"user"},
			field:       "AnyField",
			expectError: false,
		},
		{
			name:        "any field - no roles",
			roles:       []string{},
			field:       "AnyField",
			expectError: true,
		},
		{
			name:        "none field - admin role",
			roles:       []string{"admin"},
			field:       "NoneField",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRoles(context.Background(), tt.roles)
			resource := VoterTestResource{}

			err := voter.CheckWrite(ctx, resource, tt.field)

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

func TestCheckWriteValidation(t *testing.T) {
	config := DefaultConfig()
	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	ctx := context.Background()

	err = voter.CheckWrite(ctx, nil, "Field")
	if err == nil || !strings.Contains(err.Error(), "resource cannot be nil") {
		t.Error("expected error for nil resource")
	}

	resource := VoterTestResource{}
	err = voter.CheckWrite(ctx, resource, "")
	if err == nil || !strings.Contains(err.Error(), "field name cannot be empty") {
		t.Error("expected error for empty field name")
	}
}

type filterReadExpectations struct {
	expectPublic bool
	expectUser   bool
	expectAdmin  bool
	expectAny    bool
	expectNone   bool
	expectNoTag  bool
}

func validateFilterReadField(t *testing.T, fieldName, fieldValue string, expected bool) {
	if expected && fieldValue == "" {
		t.Errorf("expected %s to be visible", fieldName)
	}
	if !expected && fieldValue != "" {
		t.Errorf("expected %s to be filtered", fieldName)
	}
}

func validateFilterReadResult(t *testing.T, result VoterTestResource, exp filterReadExpectations) {
	validateFilterReadField(t, "PublicField", result.PublicField, exp.expectPublic)
	validateFilterReadField(t, "UserField", result.UserField, exp.expectUser)
	validateFilterReadField(t, "AdminField", result.AdminField, exp.expectAdmin)
	validateFilterReadField(t, "AnyField", result.AnyField, exp.expectAny)
	validateFilterReadField(t, "NoneField", result.NoneField, exp.expectNone)
	validateFilterReadField(t, "NoTagField", result.NoTagField, exp.expectNoTag)
}

func TestFilterRead(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	config.SuperuserRole = "superadmin"
	config.DefaultFieldPolicy = "deny"

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	resource := VoterTestResource{
		PublicField: "public",
		UserField:   "user",
		AdminField:  "admin",
		AnyField:    "any",
		NoneField:   "none",
		NoTagField:  "notag",
	}

	tests := []struct {
		name   string
		roles  []string
		expect filterReadExpectations
	}{
		{
			name:  "no roles",
			roles: []string{},
			expect: filterReadExpectations{
				expectPublic: true,
			},
		},
		{
			name:  "user role",
			roles: []string{"user"},
			expect: filterReadExpectations{
				expectPublic: true,
				expectUser:   true,
				expectAny:    true,
			},
		},
		{
			name:  "admin role",
			roles: []string{"admin"},
			expect: filterReadExpectations{
				expectPublic: true,
				expectUser:   true,
				expectAdmin:  true,
				expectAny:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRoles(context.Background(), tt.roles)

			filtered, err := voter.FilterRead(ctx, resource)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, ok := filtered.(VoterTestResource)
			if !ok {
				t.Fatalf("expected VoterTestResource, got %T", filtered)
			}

			validateFilterReadResult(t, result, tt.expect)
		})
	}
}

func TestFilterReadSuperuser(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	config.SuperuserRole = "admin"

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	ctx := WithRoles(context.Background(), []string{"admin"})
	resource := VoterTestResource{
		PublicField: "public",
		UserField:   "user",
		AdminField:  "admin",
		AnyField:    "any",
		NoneField:   "none",
		NoTagField:  "notag",
	}

	filtered, err := voter.FilterRead(ctx, resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := filtered.(VoterTestResource)
	if !ok {
		t.Fatalf("expected VoterTestResource, got %T", filtered)
	}

	if result.PublicField == "" || result.UserField == "" || result.AdminField == "" ||
		result.AnyField == "" || result.NoneField == "" || result.NoTagField == "" {
		t.Error("superuser should see all fields")
	}
}

func TestValidateWrite(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	tests := []struct {
		name        string
		roles       []string
		resource    VoterTestResource
		expectError bool
		errorFields []string
	}{
		{
			name:  "public field only",
			roles: []string{},
			resource: VoterTestResource{
				PublicField: "value",
			},
			expectError: false,
		},
		{
			name:  "user tries to write admin field",
			roles: []string{"user"},
			resource: VoterTestResource{
				AdminField: "value",
			},
			expectError: true,
			errorFields: []string{"AdminField"},
		},
		{
			name:  "editor writes user field",
			roles: []string{"editor"},
			resource: VoterTestResource{
				UserField: "value",
			},
			expectError: false,
		},
		{
			name:  "multiple forbidden fields",
			roles: []string{"user"},
			resource: VoterTestResource{
				AdminField: "value1",
				UserField:  "value2",
			},
			expectError: true,
			errorFields: []string{"AdminField", "UserField"},
		},
		{
			name:  "zero values are ignored",
			roles: []string{"user"},
			resource: VoterTestResource{
				AdminField: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRoles(context.Background(), tt.roles)

			err := voter.ValidateWrite(ctx, tt.resource)

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

				if len(validationErr.ForbiddenFields) != len(tt.errorFields) {
					t.Errorf("expected %d forbidden fields, got %d", len(tt.errorFields), len(validationErr.ForbiddenFields))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateWriteSuperuser(t *testing.T) {
	ClearCache()

	config := DefaultConfig()
	config.SuperuserRole = "admin"

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	ctx := WithRoles(context.Background(), []string{"admin"})
	resource := VoterTestResource{
		PublicField: "public",
		UserField:   "user",
		AdminField:  "admin",
		AnyField:    "any",
		NoneField:   "none",
		NoTagField:  "notag",
	}

	err = voter.ValidateWrite(ctx, resource)
	if err != nil {
		t.Errorf("superuser should be able to write all fields, got error: %v", err)
	}
}

func TestHandleFieldWithoutAnnotation(t *testing.T) {
	tests := []struct {
		name               string
		defaultFieldPolicy string
		expectError        bool
	}{
		{
			name:               "allow policy",
			defaultFieldPolicy: "allow",
			expectError:        false,
		},
		{
			name:               "deny policy",
			defaultFieldPolicy: "deny",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DefaultFieldPolicy = tt.defaultFieldPolicy

			voter, err := NewVoter(config)
			if err != nil {
				t.Fatalf("failed to create voter: %v", err)
			}

			dv := voter.(*defaultVoter)
			err = dv.handleFieldWithoutAnnotation("NoTagField", "read", []string{"user"})

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

func TestIsSuperuser(t *testing.T) {
	config := DefaultConfig()
	config.SuperuserRole = "admin"

	voter, err := NewVoter(config)
	if err != nil {
		t.Fatalf("failed to create voter: %v", err)
	}

	dv := voter.(*defaultVoter)

	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "has superuser role",
			roles:    []string{"admin"},
			expected: true,
		},
		{
			name:     "no superuser role",
			roles:    []string{"user"},
			expected: false,
		},
		{
			name:     "multiple roles including superuser",
			roles:    []string{"user", "admin", "editor"},
			expected: true,
		},
		{
			name:     "empty roles",
			roles:    []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dv.isSuperuser(tt.roles)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
