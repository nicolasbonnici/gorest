package processor

import (
	"fmt"
	"reflect"
)

// getModelID extracts the ID field from a model using reflection.
// It looks for a field with the "db" tag set to "id".
// Returns the ID value or an error if the ID field is not found or invalid.
func getModelID(model interface{}) (any, error) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a struct or pointer to struct")
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "id" {
			fieldValue := v.Field(i)
			if !fieldValue.IsValid() {
				return nil, fmt.Errorf("id field is invalid")
			}
			return fieldValue.Interface(), nil
		}
	}

	return nil, fmt.Errorf("id field not found in model")
}
