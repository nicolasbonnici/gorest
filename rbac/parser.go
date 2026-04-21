package rbac

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	annotationCache   = make(map[reflect.Type]PermissionSet)
	annotationCacheMu sync.RWMutex
)

func ParseAnnotations(resource interface{}) (PermissionSet, error) {
	t := reflect.TypeOf(resource)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("resource must be a struct, got %s", t.Kind())
	}

	annotationCacheMu.RLock()
	if cached, ok := annotationCache[t]; ok {
		annotationCacheMu.RUnlock()
		return cached, nil
	}
	annotationCacheMu.RUnlock()

	permissions := make(PermissionSet)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("rbac")
		if tag == "" {
			continue
		}

		perm, err := parseTag(field.Name, tag)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		permissions[field.Name] = perm
	}

	annotationCacheMu.Lock()
	if cached, ok := annotationCache[t]; ok {
		annotationCacheMu.Unlock()
		return cached, nil
	}
	annotationCache[t] = permissions
	annotationCacheMu.Unlock()

	return permissions, nil
}

// Format: "read:role1,role2;write:role3,role4"
func parseTag(fieldName, tag string) (Permission, error) {
	perm := Permission{
		Field: fieldName,
		Read:  []string{},
		Write: []string{},
	}

	parts := strings.Split(tag, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		opParts := strings.SplitN(part, ":", 2)
		if len(opParts) != 2 {
			return perm, fmt.Errorf("%w: invalid format, expected 'operation:roles', got '%s'", ErrInvalidAnnotation, part)
		}

		operation := strings.TrimSpace(opParts[0])
		rolesStr := strings.TrimSpace(opParts[1])

		var roles []string
		if rolesStr != "" {
			for _, role := range strings.Split(rolesStr, ",") {
				role = strings.TrimSpace(role)
				if role != "" {
					roles = append(roles, role)
				}
			}
		}

		switch strings.ToLower(operation) {
		case "read":
			perm.Read = roles
		case "write":
			perm.Write = roles
		default:
			return perm, fmt.Errorf("%w: unknown operation '%s', expected 'read' or 'write'", ErrInvalidAnnotation, operation)
		}
	}

	return perm, nil
}

func ClearCache() {
	annotationCacheMu.Lock()
	annotationCache = make(map[reflect.Type]PermissionSet)
	annotationCacheMu.Unlock()
}

func GetFieldNames(resource interface{}) []string {
	t := reflect.TypeOf(resource)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.IsExported() {
			fields = append(fields, field.Name)
		}
	}

	return fields
}

func HasAnnotation(resource interface{}, fieldName string) bool {
	t := reflect.TypeOf(resource)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	field, ok := t.FieldByName(fieldName)
	if !ok {
		return false
	}

	tag := field.Tag.Get("rbac")
	return tag != ""
}
