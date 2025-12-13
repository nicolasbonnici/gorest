package expand

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
)

type RelationConfig struct {
	Field        string
	ForeignKeyField string
	RelatedTable string
	CRUD interface{}
}

func ExpandRelations(ctx context.Context, data interface{}, relations []string, configs map[string]RelationConfig) (interface{}, error) {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice {
		return expandSlice(ctx, data, relations, configs)
	}
	return expandSingle(ctx, data, relations, configs)
}

func expandSingle(ctx context.Context, item interface{}, relations []string, configs map[string]RelationConfig) (interface{}, error) {
	result := make(map[string]interface{})

	itemBytes, _ := json.Marshal(item)
	json.Unmarshal(itemBytes, &result)

	for _, rel := range relations {
		cfg, ok := configs[rel]
		if !ok {
			continue
		}

		fkValue, ok := result[cfg.ForeignKeyField]
		if !ok || fkValue == nil {
			continue
		}

		fkStr, ok := fkValue.(string)
		if !ok || fkStr == "" {
			continue
		}

		relatedItem, err := fetchRelatedItem(ctx, cfg.CRUD, fkStr)
		if err != nil {
			continue
		}

		result[rel] = relatedItem
		delete(result, cfg.ForeignKeyField)
	}

	return result, nil
}

func expandSlice(ctx context.Context, items interface{}, relations []string, configs map[string]RelationConfig) (interface{}, error) {
	val := reflect.ValueOf(items)
	results := make([]interface{}, val.Len())

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()
		expandedItem, err := expandSingle(ctx, item, relations, configs)
		if err != nil {
			results[i] = item
			continue
		}
		results[i] = expandedItem
	}

	return results, nil
}

func fetchRelatedItem(ctx context.Context, crudInterface interface{}, id string) (interface{}, error) {
	method := reflect.ValueOf(crudInterface).MethodByName("GetByID")
	if !method.IsValid() {
		return nil, nil
	}

	ctxVal := reflect.ValueOf(ctx)
	idVal := reflect.ValueOf(id)

	results := method.Call([]reflect.Value{ctxVal, idVal})
	if len(results) != 2 {
		return nil, nil
	}

	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	itemPtr := results[0].Interface()
	if itemPtr == nil {
		return nil, nil
	}

	return reflect.ValueOf(itemPtr).Elem().Interface(), nil
}

func ParseExpand(expandParams []string, configs map[string]RelationConfig) []string {
	var valid []string
	for _, exp := range expandParams {
		normalized := normalizeFieldName(exp)
		if _, ok := configs[normalized]; ok {
			valid = append(valid, normalized)
		}
	}
	return valid
}

func normalizeFieldName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "_", "")
}
