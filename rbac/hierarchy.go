package rbac

import (
	"sync"
)

var (
	hierarchyCache   = make(map[string][]string)
	hierarchyCacheMu sync.RWMutex
)

func ResolveRoles(roles []string, hierarchy map[string][]string) []string {
	result, err := resolveRolesWithError(roles, hierarchy)
	if err != nil {
		return roles
	}
	return result
}

func resolveRolesWithError(roles []string, hierarchy map[string][]string) ([]string, error) {
	if len(roles) == 0 {
		return roles, nil
	}

	resolved := make(map[string]bool)

	for _, role := range roles {
		if err := expandRole(role, hierarchy, resolved, make(map[string]bool)); err != nil {
			return nil, err
		}
	}

	result := make([]string, 0, len(resolved))
	for role := range resolved {
		result = append(result, role)
	}

	return result, nil
}

func expandRole(role string, hierarchy map[string][]string, resolved map[string]bool, visited map[string]bool) error {
	if resolved[role] {
		return nil
	}

	if visited[role] {
		return ErrCircularHierarchy
	}

	visited[role] = true
	resolved[role] = true

	if children, ok := hierarchy[role]; ok {
		for _, child := range children {
			if err := expandRole(child, hierarchy, resolved, visited); err != nil {
				return err
			}
		}
	}

	delete(visited, role)
	return nil
}

func HasRole(roles []string, requiredRole string, hierarchy map[string][]string) bool {
	expanded := ResolveRoles(roles, hierarchy)
	for _, role := range expanded {
		if role == requiredRole {
			return true
		}
	}
	return false
}

func HasAnyRole(roles []string, requiredRoles []string, hierarchy map[string][]string) bool {
	expanded := ResolveRoles(roles, hierarchy)

	// Create a set of expanded roles for faster lookup
	roleSet := make(map[string]bool)
	for _, role := range expanded {
		roleSet[role] = true
	}

	// Check if any required role is in the set
	for _, required := range requiredRoles {
		if roleSet[required] {
			return true
		}
	}

	return false
}

func BuildRoleTree(hierarchy map[string][]string) ([]*RoleNode, error) {
	childRoles := make(map[string]bool)
	for _, children := range hierarchy {
		for _, child := range children {
			childRoles[child] = true
		}
	}

	var roots []*RoleNode
	for role := range hierarchy {
		if !childRoles[role] {
			node, err := buildRoleNode(role, hierarchy, make(map[string]bool))
			if err != nil {
				return nil, err
			}
			roots = append(roots, node)
		}
	}

	if len(roots) == 0 && len(hierarchy) > 0 {
		for role := range hierarchy {
			_, err := buildRoleNode(role, hierarchy, make(map[string]bool))
			if err != nil {
				return nil, err
			}
			break
		}
	}

	return roots, nil
}

func buildRoleNode(role string, hierarchy map[string][]string, visited map[string]bool) (*RoleNode, error) {
	if visited[role] {
		return nil, ErrCircularHierarchy
	}

	visited[role] = true

	node := &RoleNode{
		Name:     role,
		Children: []*RoleNode{},
	}

	if children, ok := hierarchy[role]; ok {
		for _, child := range children {
			childNode, err := buildRoleNode(child, hierarchy, visited)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, childNode)
		}
	}

	delete(visited, role)

	return node, nil
}

func ClearHierarchyCache() {
	hierarchyCacheMu.Lock()
	hierarchyCache = make(map[string][]string)
	hierarchyCacheMu.Unlock()
}
