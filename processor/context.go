package processor

import (
	"fmt"
	"reflect"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/auth"
)

type ContextEnricher func(c fiber.Ctx, model interface{}) error

func UserIDEnricher(field string) ContextEnricher {
	return func(c fiber.Ctx, model interface{}) error {
		user := auth.GetAuthenticatedUser(c)
		if user == nil {
			return nil
		}

		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Pointer {
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

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(user.UserID)
		case reflect.Pointer:
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

func TenantIDEnricher(field string) ContextEnricher {
	return func(c fiber.Ctx, model interface{}) error {
		tenantID := c.Locals("tenant_id")
		if tenantID == nil {
			return nil
		}

		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Pointer {
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

		tenantIDStr, ok := tenantID.(string)
		if !ok {
			return fmt.Errorf("tenant_id in context is not a string")
		}

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(tenantIDStr)
		case reflect.Pointer:
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
