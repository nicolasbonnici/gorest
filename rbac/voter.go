package rbac

import (
	"context"
	"fmt"
)

// Voter provides authorization checks for resources
type Voter interface {
	CheckRead(ctx context.Context, resource interface{}, field string) error
	CheckWrite(ctx context.Context, resource interface{}, field string) error
	FilterRead(ctx context.Context, resource interface{}) (interface{}, error)
	ValidateWrite(ctx context.Context, resource interface{}) error
	GetConfig() Config
	IsSuperuser(roles []string) bool
}

// defaultVoter implements the Voter interface
type defaultVoter struct {
	config Config
}

// NewVoter creates a new Voter with the given configuration
func NewVoter(config Config) (Voter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &defaultVoter{
		config: config,
	}, nil
}

// CheckRead verifies if the user has read permission for a specific field
func (v *defaultVoter) CheckRead(ctx context.Context, resource interface{}, field string) error {
	if resource == nil {
		return fmt.Errorf("resource cannot be nil")
	}

	if field == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	roles, ok := GetRoles(ctx)
	if !ok || len(roles) == 0 {
		roles = []string{}
	}

	expandedRoles := ResolveRoles(roles, v.config.RoleHierarchy)

	if v.isSuperuser(expandedRoles) {
		return nil
	}

	permissions, err := ParseAnnotations(resource)
	if err != nil {
		return fmt.Errorf("failed to parse annotations: %w", err)
	}

	perm, exists := permissions[field]
	if !exists {
		return v.handleFieldWithoutAnnotation(field, "read", expandedRoles)
	}

	if perm.HasReadPermission(expandedRoles) {
		return nil
	}

	return &FieldPermissionError{
		Field:     field,
		Operation: "read",
		Required:  perm.Read,
		UserRoles: expandedRoles,
	}
}

// CheckWrite verifies if the user has write permission for a specific field
func (v *defaultVoter) CheckWrite(ctx context.Context, resource interface{}, field string) error {
	if resource == nil {
		return fmt.Errorf("resource cannot be nil")
	}

	if field == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	roles, ok := GetRoles(ctx)
	if !ok || len(roles) == 0 {
		roles = []string{}
	}

	expandedRoles := ResolveRoles(roles, v.config.RoleHierarchy)

	if v.isSuperuser(expandedRoles) {
		return nil
	}

	permissions, err := ParseAnnotations(resource)
	if err != nil {
		return fmt.Errorf("failed to parse annotations: %w", err)
	}

	perm, exists := permissions[field]
	if !exists {
		return v.handleFieldWithoutAnnotation(field, "write", expandedRoles)
	}

	if perm.HasWritePermission(expandedRoles) {
		return nil
	}

	return &FieldPermissionError{
		Field:     field,
		Operation: "write",
		Required:  perm.Write,
		UserRoles: expandedRoles,
	}
}

// FilterRead filters a resource by removing fields the user cannot read
func (v *defaultVoter) FilterRead(ctx context.Context, resource interface{}) (interface{}, error) {
	if resource == nil {
		return nil, fmt.Errorf("resource cannot be nil")
	}

	roles, ok := GetRoles(ctx)
	if !ok || len(roles) == 0 {
		roles = []string{}
	}

	expandedRoles := ResolveRoles(roles, v.config.RoleHierarchy)

	if v.isSuperuser(expandedRoles) {
		return resource, nil
	}

	return filterReadFields(resource, expandedRoles, v.config)
}

// ValidateWrite validates that all fields in the resource can be written by the user
func (v *defaultVoter) ValidateWrite(ctx context.Context, resource interface{}) error {
	if resource == nil {
		return fmt.Errorf("resource cannot be nil")
	}

	roles, ok := GetRoles(ctx)
	if !ok || len(roles) == 0 {
		roles = []string{}
	}

	expandedRoles := ResolveRoles(roles, v.config.RoleHierarchy)

	if v.isSuperuser(expandedRoles) {
		return nil
	}

	return validateWriteFields(resource, expandedRoles, v.config)
}

// isSuperuser checks if the user has the superuser role
func (v *defaultVoter) isSuperuser(roles []string) bool {
	for _, role := range roles {
		if role == v.config.SuperuserRole {
			return true
		}
	}
	return false
}

func (v *defaultVoter) GetConfig() Config {
	return v.config
}

func (v *defaultVoter) IsSuperuser(roles []string) bool {
	expandedRoles := ResolveRoles(roles, v.config.RoleHierarchy)
	return v.isSuperuser(expandedRoles)
}

func (v *defaultVoter) handleFieldWithoutAnnotation(field, operation string, userRoles []string) error {
	if v.config.DefaultFieldPolicy == "allow" {
		return nil
	}

	return &FieldPermissionError{
		Field:     field,
		Operation: operation,
		Required:  []string{"<no annotation>"},
		UserRoles: userRoles,
	}
}
