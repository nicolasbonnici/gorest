package rbac

import (
	"fmt"
	"reflect"
)

// filterReadFields filters a resource to include only fields the user can read
func filterReadFields(resource interface{}, userRoles []string, config Config) (interface{}, error) {
	val := reflect.ValueOf(resource)
	typ := reflect.TypeOf(resource)

	isPtr := false
	if typ.Kind() == reflect.Ptr {
		isPtr = true
		val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() == reflect.Slice {
		return filterSlice(resource, userRoles, config)
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("resource must be a struct or slice, got %s", typ.Kind())
	}

	permissions, err := ParseAnnotations(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to parse annotations: %w", err)
	}

	filtered := reflect.New(typ).Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if !field.IsExported() {
			continue
		}

		fieldValue := val.Field(i)

		if canReadField(field.Name, permissions, userRoles, config) {
			filtered.Field(i).Set(fieldValue)
		} else {
			filtered.Field(i).Set(reflect.Zero(field.Type))
		}
	}

	if isPtr {
		result := reflect.New(typ)
		result.Elem().Set(filtered)
		return result.Interface(), nil
	}

	return filtered.Interface(), nil
}

// filterSlice filters a slice of resources
func filterSlice(resource interface{}, userRoles []string, config Config) (interface{}, error) {
	val := reflect.ValueOf(resource)
	typ := reflect.TypeOf(resource)

	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %s", typ.Kind())
	}

	elemType := typ.Elem()
	result := reflect.MakeSlice(typ, val.Len(), val.Cap())

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()

		filtered, err := filterReadFields(item, userRoles, config)
		if err != nil {
			return nil, fmt.Errorf("failed to filter item at index %d: %w", i, err)
		}

		filteredVal := reflect.ValueOf(filtered)

		if elemType.Kind() == reflect.Ptr && filteredVal.Kind() != reflect.Ptr {
			ptr := reflect.New(filteredVal.Type())
			ptr.Elem().Set(filteredVal)
			filteredVal = ptr
		} else if elemType.Kind() != reflect.Ptr && filteredVal.Kind() == reflect.Ptr {
			filteredVal = filteredVal.Elem()
		}

		result.Index(i).Set(filteredVal)
	}

	return result.Interface(), nil
}

// canReadField checks if a field can be read based on permissions
func canReadField(fieldName string, permissions PermissionSet, userRoles []string, config Config) bool {
	perm, exists := permissions[fieldName]
	if !exists {
		return config.DefaultFieldPolicy == "allow"
	}

	return perm.HasReadPermission(userRoles)
}
