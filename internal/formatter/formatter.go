package formatter

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// ResponseFormatter defines the interface for formatting API responses
type ResponseFormatter interface {
	Format(data interface{}) ([]byte, error)
	ContentType() string
}

// GetFormatter returns the appropriate formatter based on the format string
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

// JSONFormatter handles standard JSON serialization
type JSONFormatter struct{}

func (f *JSONFormatter) Format(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (f *JSONFormatter) ContentType() string {
	return "application/json"
}

// JSONLDFormatter handles JSON-LD serialization with schema.org context
type JSONLDFormatter struct{}

func (f *JSONLDFormatter) Format(data interface{}) ([]byte, error) {
	wrapped := f.wrapWithContext(data)
	return json.Marshal(wrapped)
}

func (f *JSONLDFormatter) ContentType() string {
	return "application/ld+json"
}

func (f *JSONLDFormatter) wrapWithContext(data interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"@context": "https://schema.org/",
	}

	// Check if data is a slice (collection)
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice {
		items := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			items[i] = f.addTypeToItem(val.Index(i).Interface())
		}
		result["@graph"] = items
		return result
	}

	// Single item
	item := f.addTypeToItem(data)
	for k, v := range item {
		result[k] = v
	}
	return result
}

func (f *JSONLDFormatter) addTypeToItem(data interface{}) map[string]interface{} {
	// Convert struct to map
	jsonBytes, _ := json.Marshal(data)
	var itemMap map[string]interface{}
	json.Unmarshal(jsonBytes, &itemMap)

	if itemMap == nil {
		itemMap = make(map[string]interface{})
	}

	// Infer @type from the data structure
	typeName := f.inferSchemaType(data)
	if typeName != "" {
		itemMap["@type"] = typeName
	}

	// Add @id if there's an id field
	if id, ok := itemMap["id"]; ok && id != nil && id != "" {
		itemMap["@id"] = fmt.Sprintf("#%v", id)
	}

	return itemMap
}

func (f *JSONLDFormatter) inferSchemaType(data interface{}) string {
	// Use the struct name directly as the @type
	typeName := reflect.TypeOf(data).Name()
	return typeName
}
