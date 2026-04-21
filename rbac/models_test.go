package rbac

import (
	"testing"
)

func TestPermissionHasReadPermission(t *testing.T) {
	tests := []struct {
		name      string
		perm      Permission
		userRoles []string
		expected  bool
	}{
		{
			name: "star keyword - any roles",
			perm: Permission{
				Read: []string{"*"},
			},
			userRoles: []string{"user"},
			expected:  true,
		},
		{
			name: "star keyword - no roles",
			perm: Permission{
				Read: []string{"*"},
			},
			userRoles: []string{},
			expected:  true,
		},
		{
			name: "any keyword - with roles",
			perm: Permission{
				Read: []string{"any"},
			},
			userRoles: []string{"user"},
			expected:  true,
		},
		{
			name: "any keyword - no roles",
			perm: Permission{
				Read: []string{"any"},
			},
			userRoles: []string{},
			expected:  false,
		},
		{
			name: "none keyword",
			perm: Permission{
				Read: []string{"none"},
			},
			userRoles: []string{"admin"},
			expected:  false,
		},
		{
			name: "specific role match",
			perm: Permission{
				Read: []string{"admin", "editor"},
			},
			userRoles: []string{"editor"},
			expected:  true,
		},
		{
			name: "specific role no match",
			perm: Permission{
				Read: []string{"admin"},
			},
			userRoles: []string{"user"},
			expected:  false,
		},
		{
			name: "multiple user roles - one matches",
			perm: Permission{
				Read: []string{"admin"},
			},
			userRoles: []string{"user", "admin", "editor"},
			expected:  true,
		},
		{
			name: "multiple allowed roles - one matches",
			perm: Permission{
				Read: []string{"admin", "editor", "moderator"},
			},
			userRoles: []string{"editor"},
			expected:  true,
		},
		{
			name: "empty read permissions",
			perm: Permission{
				Read: []string{},
			},
			userRoles: []string{"user"},
			expected:  false,
		},
		{
			name: "none mixed with roles",
			perm: Permission{
				Read: []string{"none", "admin"},
			},
			userRoles: []string{"admin"},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perm.HasReadPermission(tt.userRoles)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPermissionHasWritePermission(t *testing.T) {
	tests := []struct {
		name      string
		perm      Permission
		userRoles []string
		expected  bool
	}{
		{
			name: "star keyword - any roles",
			perm: Permission{
				Write: []string{"*"},
			},
			userRoles: []string{"user"},
			expected:  true,
		},
		{
			name: "star keyword - no roles",
			perm: Permission{
				Write: []string{"*"},
			},
			userRoles: []string{},
			expected:  true,
		},
		{
			name: "any keyword - with roles",
			perm: Permission{
				Write: []string{"any"},
			},
			userRoles: []string{"user"},
			expected:  true,
		},
		{
			name: "any keyword - no roles",
			perm: Permission{
				Write: []string{"any"},
			},
			userRoles: []string{},
			expected:  false,
		},
		{
			name: "none keyword",
			perm: Permission{
				Write: []string{"none"},
			},
			userRoles: []string{"admin"},
			expected:  false,
		},
		{
			name: "specific role match",
			perm: Permission{
				Write: []string{"admin", "editor"},
			},
			userRoles: []string{"admin"},
			expected:  true,
		},
		{
			name: "specific role no match",
			perm: Permission{
				Write: []string{"admin"},
			},
			userRoles: []string{"user"},
			expected:  false,
		},
		{
			name: "empty write permissions",
			perm: Permission{
				Write: []string{},
			},
			userRoles: []string{"user"},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.perm.HasWritePermission(tt.userRoles)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPolicyConstants(t *testing.T) {
	if DenyAll != "deny_all" {
		t.Errorf("expected DenyAll to be 'deny_all', got %s", DenyAll)
	}

	if AllowAll != "allow_all" {
		t.Errorf("expected AllowAll to be 'allow_all', got %s", AllowAll)
	}
}
