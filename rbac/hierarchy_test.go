package rbac

import (
	"reflect"
	"sort"
	"testing"
)

func TestResolveRoles(t *testing.T) {
	tests := []struct {
		name      string
		roles     []string
		hierarchy map[string][]string
		expected  []string
	}{
		{
			name:      "empty roles",
			roles:     []string{},
			hierarchy: map[string][]string{},
			expected:  []string{},
		},
		{
			name:      "no hierarchy - single role",
			roles:     []string{"user"},
			hierarchy: map[string][]string{},
			expected:  []string{"user"},
		},
		{
			name:      "no hierarchy - multiple roles",
			roles:     []string{"user", "editor"},
			hierarchy: map[string][]string{},
			expected:  []string{"user", "editor"},
		},
		{
			name:  "simple hierarchy - one level",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin": {"user"},
			},
			expected: []string{"admin", "user"},
		},
		{
			name:  "simple hierarchy - two levels",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin":  {"editor"},
				"editor": {"user"},
			},
			expected: []string{"admin", "editor", "user"},
		},
		{
			name:  "multi-level hierarchy",
			roles: []string{"superadmin"},
			hierarchy: map[string][]string{
				"superadmin": {"admin"},
				"admin":      {"editor"},
				"editor":     {"author"},
				"author":     {"user"},
			},
			expected: []string{"superadmin", "admin", "editor", "author", "user"},
		},
		{
			name:  "multiple children",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin": {"editor", "moderator", "user"},
			},
			expected: []string{"admin", "editor", "moderator", "user"},
		},
		{
			name:  "multiple roles with hierarchy",
			roles: []string{"admin", "moderator"},
			hierarchy: map[string][]string{
				"admin":     {"editor"},
				"editor":    {"user"},
				"moderator": {"user"},
			},
			expected: []string{"admin", "editor", "moderator", "user"},
		},
		{
			name:  "diamond hierarchy",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin":     {"editor", "moderator"},
				"editor":    {"user"},
				"moderator": {"user"},
			},
			expected: []string{"admin", "editor", "moderator", "user"},
		},
		{
			name:  "circular dependency - self reference",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin": {"admin"},
			},
			expected: []string{"admin"},
		},
		{
			name:  "circular dependency - two roles",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin":  {"editor"},
				"editor": {"admin"},
			},
			expected: []string{"admin", "editor"},
		},
		{
			name:  "circular dependency - three roles",
			roles: []string{"admin"},
			hierarchy: map[string][]string{
				"admin":     {"editor"},
				"editor":    {"moderator"},
				"moderator": {"admin"},
			},
			expected: []string{"admin", "editor", "moderator"},
		},
		{
			name:  "role not in hierarchy",
			roles: []string{"guest"},
			hierarchy: map[string][]string{
				"admin": {"user"},
			},
			expected: []string{"guest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveRoles(tt.roles, tt.hierarchy)

			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHasRole(t *testing.T) {
	hierarchy := map[string][]string{
		"admin":  {"editor"},
		"editor": {"user"},
	}

	tests := []struct {
		name         string
		roles        []string
		requiredRole string
		expected     bool
	}{
		{
			name:         "direct role match",
			roles:        []string{"admin"},
			requiredRole: "admin",
			expected:     true,
		},
		{
			name:         "inherited role",
			roles:        []string{"admin"},
			requiredRole: "user",
			expected:     true,
		},
		{
			name:         "middle inherited role",
			roles:        []string{"admin"},
			requiredRole: "editor",
			expected:     true,
		},
		{
			name:         "no match",
			roles:        []string{"user"},
			requiredRole: "admin",
			expected:     false,
		},
		{
			name:         "empty roles",
			roles:        []string{},
			requiredRole: "admin",
			expected:     false,
		},
		{
			name:         "multiple roles - one matches",
			roles:        []string{"user", "editor"},
			requiredRole: "editor",
			expected:     true,
		},
		{
			name:         "multiple roles - inherited match",
			roles:        []string{"guest", "admin"},
			requiredRole: "user",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasRole(tt.roles, tt.requiredRole, hierarchy)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHasAnyRole(t *testing.T) {
	hierarchy := map[string][]string{
		"admin":  {"editor"},
		"editor": {"user"},
	}

	tests := []struct {
		name          string
		roles         []string
		requiredRoles []string
		expected      bool
	}{
		{
			name:          "direct match - single required",
			roles:         []string{"admin"},
			requiredRoles: []string{"admin"},
			expected:      true,
		},
		{
			name:          "direct match - multiple required",
			roles:         []string{"editor"},
			requiredRoles: []string{"admin", "editor", "user"},
			expected:      true,
		},
		{
			name:          "inherited match",
			roles:         []string{"admin"},
			requiredRoles: []string{"user"},
			expected:      true,
		},
		{
			name:          "one of multiple required roles matches",
			roles:         []string{"admin"},
			requiredRoles: []string{"guest", "moderator", "editor"},
			expected:      true,
		},
		{
			name:          "no match",
			roles:         []string{"user"},
			requiredRoles: []string{"admin", "editor"},
			expected:      false,
		},
		{
			name:          "empty user roles",
			roles:         []string{},
			requiredRoles: []string{"admin"},
			expected:      false,
		},
		{
			name:          "empty required roles",
			roles:         []string{"admin"},
			requiredRoles: []string{},
			expected:      false,
		},
		{
			name:          "both empty",
			roles:         []string{},
			requiredRoles: []string{},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAnyRole(tt.roles, tt.requiredRoles, hierarchy)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuildRoleTree(t *testing.T) {
	tests := []struct {
		name      string
		hierarchy map[string][]string
		validate  func(*testing.T, []*RoleNode)
	}{
		{
			name:      "empty hierarchy",
			hierarchy: map[string][]string{},
			validate: func(t *testing.T, roots []*RoleNode) {
				if len(roots) != 0 {
					t.Errorf("expected 0 roots, got %d", len(roots))
				}
			},
		},
		{
			name: "single root with children",
			hierarchy: map[string][]string{
				"admin": {"editor", "user"},
			},
			validate: func(t *testing.T, roots []*RoleNode) {
				if len(roots) != 1 {
					t.Fatalf("expected 1 root, got %d", len(roots))
				}

				if roots[0].Name != "admin" {
					t.Errorf("expected root name 'admin', got %s", roots[0].Name)
				}

				if len(roots[0].Children) != 2 {
					t.Errorf("expected 2 children, got %d", len(roots[0].Children))
				}
			},
		},
		{
			name: "multiple roots",
			hierarchy: map[string][]string{
				"admin":     {"user"},
				"moderator": {"guest"},
			},
			validate: func(t *testing.T, roots []*RoleNode) {
				if len(roots) != 2 {
					t.Fatalf("expected 2 roots, got %d", len(roots))
				}
			},
		},
		{
			name: "multi-level tree",
			hierarchy: map[string][]string{
				"admin":  {"editor"},
				"editor": {"author"},
				"author": {"user"},
			},
			validate: func(t *testing.T, roots []*RoleNode) {
				if len(roots) != 1 {
					t.Fatalf("expected 1 root, got %d", len(roots))
				}

				if roots[0].Name != "admin" {
					t.Errorf("expected root name 'admin', got %s", roots[0].Name)
				}

				if len(roots[0].Children) != 1 {
					t.Fatalf("expected 1 child, got %d", len(roots[0].Children))
				}

				if roots[0].Children[0].Name != "editor" {
					t.Errorf("expected child name 'editor', got %s", roots[0].Children[0].Name)
				}
			},
		},
		{
			name: "diamond structure",
			hierarchy: map[string][]string{
				"admin":     {"editor", "moderator"},
				"editor":    {"user"},
				"moderator": {"user"},
			},
			validate: func(t *testing.T, roots []*RoleNode) {
				if len(roots) != 1 {
					t.Fatalf("expected 1 root, got %d", len(roots))
				}

				if len(roots[0].Children) != 2 {
					t.Errorf("expected 2 children at first level, got %d", len(roots[0].Children))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots, err := BuildRoleTree(tt.hierarchy)
			if err != nil {
				t.Fatalf("BuildRoleTree failed: %v", err)
			}
			tt.validate(t, roots)
		})
	}
}

func TestBuildRoleTreeCircular(t *testing.T) {
	hierarchy := map[string][]string{
		"admin":  {"editor"},
		"editor": {"admin"},
	}

	_, err := BuildRoleTree(hierarchy)
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
	if err != ErrCircularHierarchy {
		t.Errorf("expected ErrCircularHierarchy, got %v", err)
	}
}

func TestBuildRoleNode(t *testing.T) {
	hierarchy := map[string][]string{
		"admin":  {"editor"},
		"editor": {"user"},
	}

	visited := make(map[string]bool)
	node, err := buildRoleNode("admin", hierarchy, visited)
	if err != nil {
		t.Fatalf("buildRoleNode failed: %v", err)
	}

	if node.Name != "admin" {
		t.Errorf("expected node name 'admin', got %s", node.Name)
	}

	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Children))
	}

	if node.Children[0].Name != "editor" {
		t.Errorf("expected child name 'editor', got %s", node.Children[0].Name)
	}

	if len(node.Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(node.Children[0].Children))
	}

	if node.Children[0].Children[0].Name != "user" {
		t.Errorf("expected grandchild name 'user', got %s", node.Children[0].Children[0].Name)
	}
}

func TestBuildRoleNodeCircular(t *testing.T) {
	hierarchy := map[string][]string{
		"admin": {"admin"},
	}

	visited := make(map[string]bool)
	_, err := buildRoleNode("admin", hierarchy, visited)
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
	if err != ErrCircularHierarchy {
		t.Errorf("expected ErrCircularHierarchy, got %v", err)
	}
}

func TestClearHierarchyCache(t *testing.T) {
	ClearHierarchyCache()

	hierarchyCacheMu.RLock()
	cacheSize := len(hierarchyCache)
	hierarchyCacheMu.RUnlock()

	if cacheSize != 0 {
		t.Error("expected cache to be empty initially")
	}

	hierarchyCache["test"] = []string{"role1", "role2"}

	hierarchyCacheMu.RLock()
	cacheSize = len(hierarchyCache)
	hierarchyCacheMu.RUnlock()

	if cacheSize == 0 {
		t.Error("expected cache to be populated")
	}

	ClearHierarchyCache()

	hierarchyCacheMu.RLock()
	cacheSize = len(hierarchyCache)
	hierarchyCacheMu.RUnlock()

	if cacheSize != 0 {
		t.Error("expected cache to be empty after clear")
	}
}

func TestExpandRole(t *testing.T) {
	hierarchy := map[string][]string{
		"admin":  {"editor", "moderator"},
		"editor": {"user"},
	}

	resolved := make(map[string]bool)
	visited := make(map[string]bool)

	if err := expandRole("admin", hierarchy, resolved, visited); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedRoles := []string{"admin", "editor", "moderator", "user"}
	if len(resolved) != len(expectedRoles) {
		t.Errorf("expected %d roles, got %d", len(expectedRoles), len(resolved))
	}

	for _, role := range expectedRoles {
		if !resolved[role] {
			t.Errorf("expected role %s to be resolved", role)
		}
	}
}

func TestExpandRoleCircular(t *testing.T) {
	hierarchy := map[string][]string{
		"admin":  {"editor"},
		"editor": {"admin"},
	}

	resolved := make(map[string]bool)
	visited := make(map[string]bool)

	if err := expandRole("admin", hierarchy, resolved, visited); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved) != 2 {
		t.Errorf("expected 2 roles, got %d", len(resolved))
	}

	if !resolved["admin"] || !resolved["editor"] {
		t.Error("expected both admin and editor to be resolved")
	}
}
