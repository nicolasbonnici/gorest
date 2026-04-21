package rbac

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultPolicy != DenyAll {
		t.Errorf("expected default policy to be DenyAll, got %s", config.DefaultPolicy)
	}

	if config.SuperuserRole != "admin" {
		t.Errorf("expected superuser role to be 'admin', got %s", config.SuperuserRole)
	}

	if config.RoleHierarchy == nil {
		t.Error("expected role hierarchy to be initialized")
	}

	if !config.CacheEnabled {
		t.Error("expected cache to be enabled by default")
	}

	if config.CacheTTL != 300 {
		t.Errorf("expected cache TTL to be 300, got %d", config.CacheTTL)
	}

	if !config.StrictMode {
		t.Error("expected strict mode to be enabled by default")
	}

	if config.DefaultFieldPolicy != "deny" {
		t.Errorf("expected default field policy to be 'deny', got %s", config.DefaultFieldPolicy)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorField  string
	}{
		{
			name:        "valid default config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "valid config with allow_all policy",
			config: Config{
				DefaultPolicy:      AllowAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				StrictMode:         true,
				DefaultFieldPolicy: "allow",
			},
			expectError: false,
		},
		{
			name: "invalid default policy",
			config: Config{
				DefaultPolicy:      Policy("invalid"),
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				DefaultFieldPolicy: "deny",
			},
			expectError: true,
			errorField:  "default_policy",
		},
		{
			name: "empty superuser role",
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				DefaultFieldPolicy: "deny",
			},
			expectError: true,
			errorField:  "superuser_role",
		},
		{
			name: "invalid cache TTL when cache enabled",
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           0,
				DefaultFieldPolicy: "deny",
			},
			expectError: true,
			errorField:  "cache_ttl",
		},
		{
			name: "cache disabled with zero TTL is valid",
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       false,
				CacheTTL:           0,
				DefaultFieldPolicy: "deny",
			},
			expectError: false,
		},
		{
			name: "invalid default field policy",
			config: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				DefaultFieldPolicy: "invalid",
			},
			expectError: true,
			errorField:  "default_field_policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected validation error, got nil")
					return
				}

				configErr, ok := err.(*ConfigError)
				if !ok {
					t.Errorf("expected ConfigError, got %T", err)
					return
				}

				if configErr.Field != tt.errorField {
					t.Errorf("expected error field %s, got %s", tt.errorField, configErr.Field)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestConfigValidateHierarchy(t *testing.T) {
	tests := []struct {
		name        string
		hierarchy   map[string][]string
		expectError bool
	}{
		{
			name:        "empty hierarchy",
			hierarchy:   map[string][]string{},
			expectError: false,
		},
		{
			name: "simple hierarchy",
			hierarchy: map[string][]string{
				"admin":  {"editor", "user"},
				"editor": {"user"},
			},
			expectError: false,
		},
		{
			name: "multi-level hierarchy",
			hierarchy: map[string][]string{
				"superadmin": {"admin"},
				"admin":      {"editor"},
				"editor":     {"user"},
			},
			expectError: false,
		},
		{
			name: "circular dependency - direct cycle",
			hierarchy: map[string][]string{
				"admin": {"admin"},
			},
			expectError: true,
		},
		{
			name: "circular dependency - two roles",
			hierarchy: map[string][]string{
				"admin":  {"editor"},
				"editor": {"admin"},
			},
			expectError: true,
		},
		{
			name: "circular dependency - three roles",
			hierarchy: map[string][]string{
				"admin":  {"editor"},
				"editor": {"user"},
				"user":   {"admin"},
			},
			expectError: true,
		},
		{
			name: "complex hierarchy without cycles",
			hierarchy: map[string][]string{
				"admin":     {"editor", "moderator"},
				"editor":    {"author"},
				"author":    {"user"},
				"moderator": {"user"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.RoleHierarchy = tt.hierarchy

			err := config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected circular hierarchy error, got nil")
					return
				}

				configErr, ok := err.(*ConfigError)
				if !ok {
					t.Errorf("expected ConfigError, got %T", err)
					return
				}

				if configErr.Field != "role_hierarchy" {
					t.Errorf("expected error field 'role_hierarchy', got %s", configErr.Field)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestConfigMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     Config
		override Config
		expected Config
	}{
		{
			name: "merge policy",
			base: DefaultConfig(),
			override: Config{
				DefaultPolicy: AllowAll,
				CacheEnabled:  true,
				StrictMode:    true,
			},
			expected: Config{
				DefaultPolicy:      AllowAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				StrictMode:         true,
				DefaultFieldPolicy: "deny",
			},
		},
		{
			name: "merge superuser role",
			base: DefaultConfig(),
			override: Config{
				SuperuserRole: "superadmin",
				CacheEnabled:  true,
				StrictMode:    true,
			},
			expected: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "superadmin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				StrictMode:         true,
				DefaultFieldPolicy: "deny",
			},
		},
		{
			name: "merge role hierarchy",
			base: Config{
				DefaultPolicy: DenyAll,
				SuperuserRole: "admin",
				RoleHierarchy: map[string][]string{
					"admin": {"user"},
				},
				CacheEnabled:       true,
				CacheTTL:           300,
				StrictMode:         true,
				DefaultFieldPolicy: "deny",
			},
			override: Config{
				RoleHierarchy: map[string][]string{
					"editor": {"author"},
				},
				CacheEnabled: true,
			},
			expected: Config{
				DefaultPolicy: DenyAll,
				SuperuserRole: "admin",
				RoleHierarchy: map[string][]string{
					"admin":  {"user"},
					"editor": {"author"},
				},
				CacheEnabled:       true,
				CacheTTL:           300,
				StrictMode:         false,
				DefaultFieldPolicy: "deny",
			},
		},
		{
			name: "merge cache settings",
			base: DefaultConfig(),
			override: Config{
				CacheEnabled: false,
				CacheTTL:     600,
				StrictMode:   true,
			},
			expected: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       false,
				CacheTTL:           600,
				StrictMode:         true,
				DefaultFieldPolicy: "deny",
			},
		},
		{
			name: "merge default field policy",
			base: DefaultConfig(),
			override: Config{
				DefaultFieldPolicy: "allow",
				CacheEnabled:       true,
				StrictMode:         true,
			},
			expected: Config{
				DefaultPolicy:      DenyAll,
				SuperuserRole:      "admin",
				RoleHierarchy:      make(map[string][]string),
				CacheEnabled:       true,
				CacheTTL:           300,
				StrictMode:         true,
				DefaultFieldPolicy: "allow",
			},
		},
		{
			name: "merge all fields",
			base: DefaultConfig(),
			override: Config{
				DefaultPolicy: AllowAll,
				SuperuserRole: "superadmin",
				RoleHierarchy: map[string][]string{
					"superadmin": {"admin"},
				},
				CacheEnabled:       false,
				CacheTTL:           600,
				StrictMode:         false,
				DefaultFieldPolicy: "allow",
			},
			expected: Config{
				DefaultPolicy: AllowAll,
				SuperuserRole: "superadmin",
				RoleHierarchy: map[string][]string{
					"superadmin": {"admin"},
				},
				CacheEnabled:       false,
				CacheTTL:           600,
				StrictMode:         false,
				DefaultFieldPolicy: "allow",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.override)

			if tt.base.DefaultPolicy != tt.expected.DefaultPolicy {
				t.Errorf("DefaultPolicy: expected %s, got %s", tt.expected.DefaultPolicy, tt.base.DefaultPolicy)
			}

			if tt.base.SuperuserRole != tt.expected.SuperuserRole {
				t.Errorf("SuperuserRole: expected %s, got %s", tt.expected.SuperuserRole, tt.base.SuperuserRole)
			}

			if tt.base.CacheEnabled != tt.expected.CacheEnabled {
				t.Errorf("CacheEnabled: expected %v, got %v", tt.expected.CacheEnabled, tt.base.CacheEnabled)
			}

			if tt.base.CacheTTL != tt.expected.CacheTTL {
				t.Errorf("CacheTTL: expected %d, got %d", tt.expected.CacheTTL, tt.base.CacheTTL)
			}

			if tt.base.StrictMode != tt.expected.StrictMode {
				t.Errorf("StrictMode: expected %v, got %v", tt.expected.StrictMode, tt.base.StrictMode)
			}

			if tt.base.DefaultFieldPolicy != tt.expected.DefaultFieldPolicy {
				t.Errorf("DefaultFieldPolicy: expected %s, got %s", tt.expected.DefaultFieldPolicy, tt.base.DefaultFieldPolicy)
			}

			for key, expectedRoles := range tt.expected.RoleHierarchy {
				actualRoles, ok := tt.base.RoleHierarchy[key]
				if !ok {
					t.Errorf("RoleHierarchy: expected key %s to exist", key)
					continue
				}

				if len(actualRoles) != len(expectedRoles) {
					t.Errorf("RoleHierarchy[%s]: expected %d roles, got %d", key, len(expectedRoles), len(actualRoles))
					continue
				}

				for i, expectedRole := range expectedRoles {
					if actualRoles[i] != expectedRole {
						t.Errorf("RoleHierarchy[%s][%d]: expected %s, got %s", key, i, expectedRole, actualRoles[i])
					}
				}
			}
		})
	}
}
