package rbac

import (
	"errors"
	"strings"
	"testing"
)

func TestFieldPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldPermissionError
		expected string
	}{
		{
			name: "basic permission error",
			err: &FieldPermissionError{
				Field:     "Email",
				Operation: "read",
				Required:  []string{"admin", "editor"},
				UserRoles: []string{"user"},
			},
			expected: "permission denied for field 'Email' (read): requires one of [admin editor], user has [user]",
		},
		{
			name: "write operation",
			err: &FieldPermissionError{
				Field:     "Password",
				Operation: "write",
				Required:  []string{"admin"},
				UserRoles: []string{"user", "editor"},
			},
			expected: "permission denied for field 'Password' (write): requires one of [admin], user has [user editor]",
		},
		{
			name: "no user roles",
			err: &FieldPermissionError{
				Field:     "Secret",
				Operation: "read",
				Required:  []string{"admin"},
				UserRoles: []string{},
			},
			expected: "permission denied for field 'Secret' (read): requires one of [admin], user has []",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "single forbidden field",
			err: &ValidationError{
				ForbiddenFields: []string{"AdminField"},
			},
			expected: "forbidden field: AdminField",
		},
		{
			name: "multiple forbidden fields",
			err: &ValidationError{
				ForbiddenFields: []string{"AdminField", "SecretField", "PrivateField"},
			},
			expected: "forbidden fields: [AdminField SecretField PrivateField]",
		},
		{
			name: "two forbidden fields",
			err: &ValidationError{
				ForbiddenFields: []string{"Field1", "Field2"},
			},
			expected: "forbidden fields: [Field1 Field2]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		expected string
	}{
		{
			name: "basic config error",
			err: &ConfigError{
				Field:   "default_policy",
				Message: "must be 'deny_all' or 'allow_all'",
			},
			expected: "config error in 'default_policy': must be 'deny_all' or 'allow_all'",
		},
		{
			name: "circular hierarchy error",
			err: &ConfigError{
				Field:   "role_hierarchy",
				Message: "circular dependency detected involving role 'admin'",
			},
			expected: "config error in 'role_hierarchy': circular dependency detected involving role 'admin'",
		},
		{
			name: "empty superuser role error",
			err: &ConfigError{
				Field:   "superuser_role",
				Message: "cannot be empty",
			},
			expected: "config error in 'superuser_role': cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrPermissionDenied",
			err:      ErrPermissionDenied,
			expected: "permission denied",
		},
		{
			name:     "ErrInvalidAnnotation",
			err:      ErrInvalidAnnotation,
			expected: "invalid rbac annotation",
		},
		{
			name:     "ErrInvalidConfig",
			err:      ErrInvalidConfig,
			expected: "invalid configuration",
		},
		{
			name:     "ErrRoleNotFound",
			err:      ErrRoleNotFound,
			expected: "role not found",
		},
		{
			name:     "ErrUserNotFound",
			err:      ErrUserNotFound,
			expected: "user not found",
		},
		{
			name:     "ErrCircularHierarchy",
			err:      ErrCircularHierarchy,
			expected: "circular role hierarchy detected",
		},
		{
			name:     "ErrEmptyRoles",
			err:      ErrEmptyRoles,
			expected: "no roles provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorsAreComparable(t *testing.T) {
	err1 := ErrPermissionDenied
	err2 := ErrPermissionDenied

	if !errors.Is(err1, err2) {
		t.Error("expected errors to be comparable with errors.Is")
	}

	if err1 != err2 {
		t.Error("expected error constants to be the same")
	}
}

func TestFieldPermissionErrorImplementsError(t *testing.T) {
	var _ error = &FieldPermissionError{}
}

func TestValidationErrorImplementsError(t *testing.T) {
	var _ error = &ValidationError{}
}

func TestConfigErrorImplementsError(t *testing.T) {
	var _ error = &ConfigError{}
}

func TestErrorFormatting(t *testing.T) {
	fpErr := &FieldPermissionError{
		Field:     "TestField",
		Operation: "read",
		Required:  []string{"admin"},
		UserRoles: []string{"user"},
	}

	if !strings.Contains(fpErr.Error(), "TestField") {
		t.Error("error message should contain field name")
	}

	if !strings.Contains(fpErr.Error(), "read") {
		t.Error("error message should contain operation")
	}

	if !strings.Contains(fpErr.Error(), "admin") {
		t.Error("error message should contain required roles")
	}

	if !strings.Contains(fpErr.Error(), "user") {
		t.Error("error message should contain user roles")
	}
}
