package processor

import (
	"fmt"
	"reflect"
)

func getModelID(model interface{}) (any, error) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
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
