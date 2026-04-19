package rbac

import (
	"fmt"
	"reflect"
)

// validateWriteFields validates that all non-zero fields in the resource can be written by the user.
// If config.StrictValidation is true, ALL fields (including zero values) are validated.
func validateWriteFields(resource interface{}, userRoles []string, config Config) error {
	val := reflect.ValueOf(resource)
	typ := reflect.TypeOf(resource)

	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("resource must be a struct, got %s", typ.Kind())
	}

	permissions, err := ParseAnnotations(resource)
	if err != nil {
		return fmt.Errorf("failed to parse annotations: %w", err)
	}

	var forbiddenFields []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if !field.IsExported() {
			continue
		}

		fieldValue := val.Field(i)

		if !config.StrictValidation && isZeroValue(fieldValue) {
			continue
		}

		if !canWriteField(field.Name, permissions, userRoles, config) {
			forbiddenFields = append(forbiddenFields, field.Name)
		}
	}

	if len(forbiddenFields) > 0 {
		return &ValidationError{
			ForbiddenFields: forbiddenFields,
		}
	}

	return nil
}

// canWriteField checks if a field can be written based on permissions
func canWriteField(fieldName string, permissions PermissionSet, userRoles []string, config Config) bool {
	perm, exists := permissions[fieldName]
	if !exists {
		return config.DefaultFieldPolicy == "allow"
	}

	return perm.HasWritePermission(userRoles)
}

// isZeroValue checks if a reflect.Value is the zero value for its type
func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		return isStructZero(v)
	default:
		return false
	}
}

// isStructZero checks if all fields in a struct are zero values
func isStructZero(v reflect.Value) bool {
	for i := 0; i < v.NumField(); i++ {
		if !isZeroValue(v.Field(i)) {
			return false
		}
	}
	return true
}
