package serializer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type ResponseSerializer interface {
	Serialize(data interface{}, path string) ([]byte, error)
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

func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

type JSONLDSerializer struct{}

func (s *JSONLDSerializer) Serialize(data interface{}, path string) ([]byte, error) {
	wrapped := s.wrapWithContext(data, path)
	return json.Marshal(wrapped)
}

func (s *JSONLDSerializer) ContentType() string {
	return "application/ld+json"
}

func (s *JSONLDSerializer) wrapWithContext(data interface{}, path string) map[string]interface{} {
	result := map[string]interface{}{
		"@context": "https://schema.org/",
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice {
		items := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			items[i] = s.addTypeToItem(val.Index(i).Interface(), path)
		}
		result["@graph"] = items
		return result
	}

	item := s.addTypeToItem(data, path)
	for k, v := range item {
		result[k] = v
	}
	return result
}

func (s *JSONLDSerializer) AddTypeToItemExported(data interface{}, path string) map[string]interface{} {
	return s.addTypeToItem(data, path)
}

func (s *JSONLDSerializer) addTypeToItem(data interface{}, path string) map[string]interface{} {
	jsonBytes, _ := json.Marshal(data)
	var itemMap map[string]interface{}
	json.Unmarshal(jsonBytes, &itemMap)

	if itemMap == nil {
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
