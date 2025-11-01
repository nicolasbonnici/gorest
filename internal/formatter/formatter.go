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

	for key, value := range itemMap {
		isForeignKey := (strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "Id")) && key != "id"
		if isForeignKey {
			if valueStr, ok := value.(string); ok && valueStr != "" {
				suffix := "_id"
				if strings.HasSuffix(key, "Id") {
					suffix = "Id"
				}
				resourceName := pluralize(strings.TrimSuffix(key, suffix))
				itemMap[key] = fmt.Sprintf("/%s/%s", resourceName, valueStr)
			}
		}
	}

	return itemMap
}

func (f *JSONLDFormatter) inferSchemaType(data interface{}) string {
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
