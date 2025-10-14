package formatter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type ResponseFormatter interface {
	Format(data interface{}, path string) ([]byte, error)
	ContentType() string
}

func GetFormatter(format string) ResponseFormatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "jsonld", "json-ld":
		return &JSONLDFormatter{}
	default:
		// JSON-LD is the default
		return &JSONLDFormatter{}
	}
}

type JSONFormatter struct{}

func (f *JSONFormatter) Format(data interface{}, path string) ([]byte, error) {
	return json.Marshal(data)
}

func (f *JSONFormatter) ContentType() string {
	return "application/json"
}

type JSONLDFormatter struct{}

func (f *JSONLDFormatter) Format(data interface{}, path string) ([]byte, error) {
	wrapped := f.wrapWithContext(data, path)
	return json.Marshal(wrapped)
}

func (f *JSONLDFormatter) ContentType() string {
	return "application/ld+json"
}

func (f *JSONLDFormatter) wrapWithContext(data interface{}, path string) map[string]interface{} {
	result := map[string]interface{}{
		"@context": "https://schema.org/",
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice {
		items := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			items[i] = f.addTypeToItem(val.Index(i).Interface(), path)
		}
		result["@graph"] = items
		return result
	}

	item := f.addTypeToItem(data, path)
	for k, v := range item {
		result[k] = v
	}
	return result
}

func (f *JSONLDFormatter) addTypeToItem(data interface{}, path string) map[string]interface{} {
	jsonBytes, _ := json.Marshal(data)
	var itemMap map[string]interface{}
	json.Unmarshal(jsonBytes, &itemMap)

	if itemMap == nil {
		itemMap = make(map[string]interface{})
	}

	typeName := f.inferSchemaType(data)
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

	return itemMap
}

func (f *JSONLDFormatter) inferSchemaType(data interface{}) string {
	return reflect.TypeOf(data).Name()
}
