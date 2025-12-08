package codegen

import "strings"

func GoTypeToOpenAPIType(goType string) (string, string) {
	goType = strings.TrimPrefix(goType, "*")

	typeMap := map[string]struct{ typ, format string }{
		"int":         {"integer", "int32"},
		"int32":       {"integer", "int32"},
		"int64":       {"integer", "int64"},
		"int16":       {"integer", "int32"},
		"float32":     {"number", "float"},
		"float64":     {"number", "double"},
		"string":      {"string", ""},
		"bool":        {"boolean", ""},
		"time.Time":   {"string", "date-time"},
		"interface{}": {"object", ""},
	}

	if mapping, ok := typeMap[goType]; ok {
		return mapping.typ, mapping.format
	}
	return "string", ""
}

func Pluralize(word string) string {
	if strings.HasSuffix(word, "y") && !isVowel(word[len(word)-2]) {
		return word[:len(word)-1] + "ies"
	}
	if strings.HasSuffix(word, "fe") {
		return word[:len(word)-2] + "ves"
	}
	if strings.HasSuffix(word, "f") {
		return word[:len(word)-1] + "ves"
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
