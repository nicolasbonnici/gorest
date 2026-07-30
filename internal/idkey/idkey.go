// Package idkey renders resource identifiers as map keys.
//
// Relation expansion keys fetched rows on one side and foreign key values on
// the other, in different packages. Both go through Format so the two agree
// whatever the underlying id type is (string, uuid.UUID, integer).
package idkey

import (
	"fmt"
	"reflect"
	"strconv"
)

// Format reports false for identifiers that cannot address a row: nil, the
// empty string, and types with no meaningful textual form.
func Format(v any) (string, bool) {
	switch id := v.(type) {
	case nil:
		return "", false
	case string:
		return id, id != ""
	case float64:
		// JSON decoding turns every number into a float64.
		return strconv.FormatFloat(id, 'f', -1, 64), true
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return "", false
	}

	// Checked before dereferencing so types with a pointer receiver still match.
	if s, ok := v.(fmt.Stringer); ok {
		str := s.String()
		return str, str != ""
	}

	switch rv.Kind() {
	case reflect.Pointer:
		return Format(rv.Elem().Interface())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), true
	}

	return "", false
}
