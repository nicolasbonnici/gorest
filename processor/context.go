package processor

import (
	"fmt"
	"reflect"

	"github.com/gofiber/fiber/v2"
	auth "github.com/nicolasbonnici/gorest-auth"
)

// ContextEnricher is a function that enriches a model with data from the request context.
// Common use cases: auto-populate user_id from auth, tenant_id from context, etc.
type ContextEnricher func(c *fiber.Ctx, model interface{}) error

// UserIDEnricher creates a ContextEnricher that populates a field with the authenticated user's ID.
// The field parameter should be the name of the struct field to populate (e.g., "UserId").
// This enricher uses the gorest-auth plugin to extract the authenticated user.
func UserIDEnricher(field string) ContextEnricher {
	return func(c *fiber.Ctx, model interface{}) error {
		user := auth.GetAuthenticatedUser(c)
		if user == nil {
			// No authenticated user, skip enrichment
			return nil
		}

		// Use reflection to set the field value
		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if !v.IsValid() || v.Kind() != reflect.Struct {
			return fmt.Errorf("model must be a struct or pointer to struct")
		}

		fieldValue := v.FieldByName(field)
		if !fieldValue.IsValid() {
			return fmt.Errorf("field %s not found in model", field)
		}

		if !fieldValue.CanSet() {
			return fmt.Errorf("field %s cannot be set", field)
		}

		// Set the user ID based on the field type
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(user.UserID)
		case reflect.Ptr:
			// Handle *string
			if fieldValue.Type().Elem().Kind() == reflect.String {
				userIDCopy := user.UserID
				fieldValue.Set(reflect.ValueOf(&userIDCopy))
			} else {
				return fmt.Errorf("field %s has unsupported pointer type", field)
			}
		default:
			return fmt.Errorf("field %s must be string or *string", field)
		}

		return nil
	}
}

// TenantIDEnricher creates a ContextEnricher that populates a field with a tenant ID from context.
// The field parameter should be the name of the struct field to populate (e.g., "TenantId").
// The tenant ID is expected to be stored in the Fiber context under the "tenant_id" key.
func TenantIDEnricher(field string) ContextEnricher {
	return func(c *fiber.Ctx, model interface{}) error {
		// Get tenant_id from context
		tenantID := c.Locals("tenant_id")
		if tenantID == nil {
			// No tenant ID in context, skip enrichment
			return nil
		}

		// Use reflection to set the field value
		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if !v.IsValid() || v.Kind() != reflect.Struct {
			return fmt.Errorf("model must be a struct or pointer to struct")
		}

		fieldValue := v.FieldByName(field)
		if !fieldValue.IsValid() {
			return fmt.Errorf("field %s not found in model", field)
		}

		if !fieldValue.CanSet() {
			return fmt.Errorf("field %s cannot be set", field)
		}

		// Set the tenant ID based on the field type
		tenantIDStr, ok := tenantID.(string)
		if !ok {
			return fmt.Errorf("tenant_id in context is not a string")
		}

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(tenantIDStr)
		case reflect.Ptr:
			// Handle *string
			if fieldValue.Type().Elem().Kind() == reflect.String {
				tenantIDCopy := tenantIDStr
				fieldValue.Set(reflect.ValueOf(&tenantIDCopy))
			} else {
				return fmt.Errorf("field %s has unsupported pointer type", field)
			}
		default:
			return fmt.Errorf("field %s must be string or *string", field)
		}

		return nil
	}
}
