package rbac

import (
	"errors"
	"fmt"
)

var (
	ErrPermissionDenied  = errors.New("permission denied")
	ErrInvalidAnnotation = errors.New("invalid rbac annotation")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrRoleNotFound      = errors.New("role not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrCircularHierarchy = errors.New("circular role hierarchy detected")
	ErrEmptyRoles        = errors.New("no roles provided")
)

// FieldPermissionError represents a permission error for a specific field
type FieldPermissionError struct {
	Field     string
	Operation string // "read" or "write"
	Required  []string
	UserRoles []string
}

func (e *FieldPermissionError) Error() string {
	return fmt.Sprintf(
		"permission denied for field '%s' (%s): requires one of %v, user has %v",
		e.Field,
		e.Operation,
		e.Required,
		e.UserRoles,
	)
}

// ValidationError represents multiple field validation errors
type ValidationError struct {
	ForbiddenFields []string
}

func (e *ValidationError) Error() string {
	if len(e.ForbiddenFields) == 1 {
		return fmt.Sprintf("forbidden field: %s", e.ForbiddenFields[0])
	}
	return fmt.Sprintf("forbidden fields: %v", e.ForbiddenFields)
}

// ConfigError represents a configuration error with details
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error in '%s': %s", e.Field, e.Message)
}
