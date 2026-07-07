package serializer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// structField caches json tag metadata for a single exported field.
type structField struct {
	jsonKey   string
	index     int
	omitempty bool
}

type structMeta struct {
	fields []structField
}

var structCache sync.Map // map[reflect.Type]*structMeta

var marshalerType = reflect.TypeFor[json.Marshaler]()

// hasCustomJSON reports whether t (or its pointer) implements json.Marshaler.
// Such types (e.g. time.Time) must be left intact so their MarshalJSON runs,
// rather than being decomposed field-by-field into a map.
func hasCustomJSON(t reflect.Type) bool {
	return t.Implements(marshalerType) || reflect.PointerTo(t).Implements(marshalerType)
}

func getStructMeta(t reflect.Type) *structMeta {
	if v, ok := structCache.Load(t); ok {
		return v.(*structMeta)
	}
	meta := &structMeta{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		key := f.Name
		omitempty := false
		if tag != "" {
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] != "" {
				key = parts[0]
			}
			if len(parts) > 1 && strings.Contains(parts[1], "omitempty") {
				omitempty = true
			}
		}
		meta.fields = append(meta.fields, structField{jsonKey: key, index: i, omitempty: omitempty})
	}
	structCache.Store(t, meta)
	return meta
}

// structToMap converts a struct to map[string]interface{} using json tags,
// avoiding a json.Marshal/Unmarshal round-trip.
func structToMap(data interface{}) map[string]interface{} {
	val := reflect.ValueOf(data)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	if m, ok := data.(map[string]interface{}); ok {
		return m
	}
	if val.Kind() != reflect.Struct {
		// Fallback for uncommon types (slices, maps with non-string keys, etc.)
		b, _ := json.Marshal(data)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		return m
	}

	meta := getStructMeta(val.Type())
	result := make(map[string]interface{}, len(meta.fields))
	for _, f := range meta.fields {
		fv := val.Field(f.index)
		if f.omitempty && fv.IsZero() {
			continue
		}
		switch fv.Kind() {
		case reflect.Struct:
			if hasCustomJSON(fv.Type()) {
				result[f.jsonKey] = fv.Interface()
			} else {
				result[f.jsonKey] = structToMap(fv.Interface())
			}
		case reflect.Ptr:
			if !fv.IsNil() && fv.Elem().Kind() == reflect.Struct && !hasCustomJSON(fv.Elem().Type()) {
				result[f.jsonKey] = structToMap(fv.Interface())
			} else {
				result[f.jsonKey] = fv.Interface()
			}
		default:
			result[f.jsonKey] = fv.Interface()
		}
	}
	return result
}

type ResponseSerializer interface {
	Serialize(data interface{}, path string) ([]byte, error)
	SerializeWithExpand(data interface{}, path string, expand []string) ([]byte, error)
	ContentType() string
}

func GetSerializer(format string) ResponseSerializer {
	switch format {
	case "json":
		return &JSONSerializer{}
	case "jsonld", "json-ld":
		return &JSONLDSerializer{}
	default:
		// JSON-LD is the default
		return &JSONLDSerializer{}
	}
}

type JSONSerializer struct{}

func (s *JSONSerializer) Serialize(data interface{}, path string) ([]byte, error) {
	return json.Marshal(data)
}

func (s *JSONSerializer) SerializeWithExpand(data interface{}, path string, expand []string) ([]byte, error) {
	// JSON serializer doesn't do IRI conversion, so expand has no effect
	return s.Serialize(data, path)
}

func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

type JSONLDSerializer struct{}

func (s *JSONLDSerializer) Serialize(data interface{}, path string) ([]byte, error) {
	return s.SerializeWithExpand(data, path, nil)
}

func (s *JSONLDSerializer) SerializeWithExpand(data interface{}, path string, expand []string) ([]byte, error) {
	// Fast streaming path for the common case (no relation expansion). Falls
	// back to the map path for anything it can't faithfully reproduce.
	if len(expand) == 0 {
		if out, ok := fastSerializeLD(data, path); ok {
			return out, nil
		}
	}
	wrapped := s.wrapWithContextExpand(data, path, expand)
	return json.Marshal(wrapped)
}

func (s *JSONLDSerializer) ContentType() string {
	return "application/ld+json"
}

func (s *JSONLDSerializer) wrapWithContext(data interface{}, path string) map[string]interface{} {
	return s.wrapWithContextExpand(data, path, nil)
}

func (s *JSONLDSerializer) wrapWithContextExpand(data interface{}, path string, expand []string) map[string]interface{} {
	result := map[string]interface{}{
		"@context": "https://schema.org/",
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice {
		items := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			items[i] = s.addTypeToItemExpand(val.Index(i).Interface(), path, expand)
		}
		result["@graph"] = items
		return result
	}

	item := s.addTypeToItemExpand(data, path, expand)
	for k, v := range item {
		result[k] = v
	}
	return result
}

func (s *JSONLDSerializer) AddTypeToItemExported(data interface{}, path string) map[string]interface{} {
	return s.addTypeToItem(data, path)
}

func (s *JSONLDSerializer) addTypeToItem(data interface{}, path string) map[string]interface{} {
	return s.addTypeToItemExpand(data, path, nil)
}

func (s *JSONLDSerializer) addTypeToItemExpand(data interface{}, path string, expand []string) map[string]interface{} {
	raw := structToMap(data)
	// Work on a copy so we don't mutate caller-owned maps.
	itemMap := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		itemMap[k] = v
	}
	if len(itemMap) == 0 {
		itemMap = make(map[string]interface{})
	}

	typeName := s.inferSchemaType(data)
	if typeName != "" {
		itemMap["@type"] = typeName
	}

	if id, ok := itemMap["id"]; ok && id != nil && id != "" {
		idStr := fmt.Sprintf("%v", id)
		cleanPath := strings.TrimSuffix(path, "/")

		if !strings.HasSuffix(cleanPath, idStr) {
			itemMap["@id"] = fmt.Sprintf("%s/%s", cleanPath, idStr)
		} else {
			itemMap["@id"] = cleanPath
		}
	}

	expandMap := make(map[string]bool)
	for _, exp := range expand {
		expandMap[exp] = true
		expandMap[toSnakeCase(exp)] = true
		expandMap[toCamelCase(exp)] = true
	}

	for key, value := range itemMap {
		if expandMap[key] {
			if objMap, ok := value.(map[string]interface{}); ok {
				itemMap[key] = s.addTypeToItemExpand(objMap, "/"+pluralize(key), nil)
			}
		}

		isForeignKey := (strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "Id")) && key != "id"
		if isForeignKey {
			if valueStr, ok := value.(string); ok && valueStr != "" {
				suffix := "_id"
				if strings.HasSuffix(key, "Id") {
					suffix = "Id"
				}
				relationName := strings.TrimSuffix(key, suffix)
				resourceName := pluralize(relationName)

				itemMap[relationName] = fmt.Sprintf("/%s/%s", resourceName, valueStr)
				delete(itemMap, key)
			}
		}
	}

	return itemMap
}

func (s *JSONLDSerializer) inferSchemaType(data interface{}) string {
	return reflect.TypeOf(data).Name()
}

func pluralize(word string) string {
	if strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(word[len(word)-2]) {
		return word[:len(word)-1] + "ies"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}
	return word + "s"
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if i > 0 && len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

type RelationConfig struct {
	Field           string
	ForeignKeyField string
	RelatedTable    string
	CRUD            interface{}
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
