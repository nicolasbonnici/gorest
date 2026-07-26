package serializer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// fastld is a streaming JSON-LD encoder for the common case: a struct (or slice
// of structs) with no expand relations. It writes each row straight into a
// reused buffer, skipping the per-row map[string]interface{} build, the map
// copy, and the map-based json.Marshal (which sorts keys and boxes every field)
// that dominate the reflect/allocation cost of the map path. Anything it can't
// handle (expand requested, non-struct data, unexpected kinds) returns
// handled=false so the caller falls back to the original map path, which
// remains the source of truth for behavior.

type ldField struct {
	jsonKey   string
	index     int
	omitempty bool
	// isStringFK marks a `<name>_id` / `<name>Id` string field (never "id").
	// The map path only rewrites string-typed foreign keys to IRIs; uuid.UUID
	// or numeric keys are left as-is, so we mirror that exactly.
	isStringFK   bool
	relationName string // e.g. "author" for author_id
	resourceName string // pluralized, e.g. "authors"
}

type ldPlan struct {
	typeName     string
	idFieldIndex int // -1 if the struct has no json "id" field
	fields       []ldField
	// simple is true when every field is directly streamable (scalar, or a
	// type json.Marshal handles leaf-style such as time.Time/uuid). Nested
	// structs/pointers/slices/maps still stream via json.Marshal per value,
	// which is the fast cached struct encoder — also fine.
}

var ldPlanCache sync.Map // map[reflect.Type]*ldPlan

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func getLDPlan(t reflect.Type) *ldPlan {
	if v, ok := ldPlanCache.Load(t); ok {
		return v.(*ldPlan)
	}
	p := &ldPlan{typeName: t.Name(), idFieldIndex: -1}
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
		fld := ldField{jsonKey: key, index: i, omitempty: omitempty}
		if key == "id" {
			p.idFieldIndex = i
		}
		if key != "id" && f.Type.Kind() == reflect.String {
			if suffix := fkSuffix(key); suffix != "" {
				rel := strings.TrimSuffix(key, suffix)
				fld.isStringFK = true
				fld.relationName = rel
				fld.resourceName = pluralize(rel)
			}
		}
		p.fields = append(p.fields, fld)
	}
	ldPlanCache.Store(t, p)
	return p
}

func fkSuffix(key string) string {
	switch {
	case strings.HasSuffix(key, "_id"):
		return "_id"
	case strings.HasSuffix(key, "Id"):
		return "Id"
	}
	return ""
}

// fastSerializeLD tries to stream data as JSON-LD. It returns (bytes, true) on
// success, or (nil, false) when the caller should use the map-based path.
func fastSerializeLD(data interface{}, path string) ([]byte, bool) {
	val := reflect.ValueOf(data)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, false
		}
		val = val.Elem()
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	buf.WriteString(`{"@context":"https://schema.org/"`)

	switch val.Kind() {
	case reflect.Slice:
		et := val.Type().Elem()
		for et.Kind() == reflect.Ptr {
			et = et.Elem()
		}
		if et.Kind() != reflect.Struct || hasCustomJSON(et) {
			return nil, false
		}
		plan := getLDPlan(et)
		buf.WriteString(`,"@graph":[`)
		for i := 0; i < val.Len(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			iv := val.Index(i)
			for iv.Kind() == reflect.Ptr {
				if iv.IsNil() {
					break
				}
				iv = iv.Elem()
			}
			if !writeLDItem(buf, iv, path, plan) {
				return nil, false
			}
		}
		buf.WriteByte(']')
	case reflect.Struct:
		if hasCustomJSON(val.Type()) {
			return nil, false
		}
		plan := getLDPlan(val.Type())
		// Single item: @context and the item fields live in the same object.
		if !writeLDItemInline(buf, val, path, plan) {
			return nil, false
		}
	default:
		return nil, false
	}

	buf.WriteByte('}')
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, true
}

// writeLDItem writes a full `{...}` object for one graph member.
func writeLDItem(buf *bytes.Buffer, val reflect.Value, path string, plan *ldPlan) bool {
	buf.WriteByte('{')
	if !writeLDBody(buf, val, path, plan, false) {
		return false
	}
	buf.WriteByte('}')
	return true
}

// writeLDItemInline writes item fields into the already-open @context object,
// so it always emits a leading comma before the first field it writes.
func writeLDItemInline(buf *bytes.Buffer, val reflect.Value, path string, plan *ldPlan) bool {
	return writeLDBody(buf, val, path, plan, true)
}

// writeLDBody emits @type, @id and every field. hasLeading indicates a field
// (@context) was already written, so a comma must precede the first emission.
func writeLDBody(buf *bytes.Buffer, val reflect.Value, path string, plan *ldPlan, hasLeading bool) bool {
	first := !hasLeading
	sep := func() {
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
	}

	if plan.typeName != "" {
		sep()
		buf.WriteString(`"@type":`)
		writeJSONString(buf, plan.typeName)
	}

	// @id derived from the "id" field, matching the map path's rules. Renders
	// into a stack array, not buf.AvailableBuffer, which the writes below would
	// overwrite.
	if plan.idFieldIndex >= 0 {
		var idScratch [64]byte
		idBytes, ok := appendIDText(idScratch[:0], val.Field(plan.idFieldIndex))
		if ok && len(idBytes) > 0 {
			cleanPath := strings.TrimSuffix(path, "/")
			sep()
			buf.WriteString(`"@id":`)
			buf.WriteByte('"')
			writeJSONStringBody(buf, cleanPath)
			if !hasSuffixBytes(cleanPath, idBytes) {
				buf.WriteByte('/')
				writeJSONStringBodyBytes(buf, idBytes)
			}
			buf.WriteByte('"')
		}
	}

	for i := range plan.fields {
		f := &plan.fields[i]
		fv := val.Field(f.index)
		if f.omitempty && fv.IsZero() {
			continue
		}
		if f.isStringFK {
			s := fv.String()
			if s == "" {
				// Map path leaves an empty string FK untouched (still emitted
				// under its original key), so fall back to preserve behavior.
				return false
			}
			sep()
			writeJSONString(buf, f.relationName)
			buf.WriteByte(':')
			writeJSONString(buf, "/"+f.resourceName+"/"+s)
			continue
		}
		sep()
		writeJSONString(buf, f.jsonKey)
		buf.WriteByte(':')
		if !writeJSONValue(buf, fv) {
			return false
		}
	}
	return true
}

// writeJSONValue writes scalars directly (zero alloc) and defers anything else
// to json.Marshal, which uses the cached struct/leaf encoders.
func writeJSONValue(buf *bytes.Buffer, fv reflect.Value) bool {
	switch fv.Kind() {
	case reflect.String:
		writeJSONString(buf, fv.String())
	case reflect.Bool:
		if fv.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), fv.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), fv.Uint(), 10))
	default:
		// Both are wider than a pointer, so fv.Interface() heap-allocates a copy
		// per field. Addressable values (any slice element, i.e. the list path)
		// go through a *T, which boxes into the interface word for free.
		switch fv.Type() {
		case timeType:
			var tv time.Time
			if fv.CanAddr() {
				tv = *fv.Addr().Interface().(*time.Time)
			} else {
				tv = fv.Interface().(time.Time)
			}
			if y := tv.Year(); y >= 0 && y <= 9999 {
				// Matches time.Time.MarshalJSON (RFC3339Nano, quoted).
				buf.WriteByte('"')
				buf.Write(tv.AppendFormat(buf.AvailableBuffer(), time.RFC3339Nano))
				buf.WriteByte('"')
				return true
			}
		case uuidType:
			// Matches uuid's MarshalText: canonical lower-case, quoted.
			buf.WriteByte('"')
			if fv.CanAddr() {
				buf.Write(appendUUID(buf.AvailableBuffer(), fv.Addr().Interface().(*uuid.UUID)))
			} else {
				u := fv.Interface().(uuid.UUID)
				buf.Write(appendUUID(buf.AvailableBuffer(), &u))
			}
			buf.WriteByte('"')
			return true
		}
		b, err := json.Marshal(fv.Interface())
		if err != nil {
			return false
		}
		buf.Write(b)
	}
	return true
}

var (
	timeType = reflect.TypeOf(time.Time{})
	uuidType = reflect.TypeOf(uuid.UUID{})
)

// appendUUID matches uuid.UUID.String() without its string allocation.
func appendUUID(dst []byte, u *uuid.UUID) []byte {
	for i, b := range u {
		dst = append(dst, hexDigit[b>>4], hexDigit[b&0xF])
		if i == 3 || i == 5 || i == 7 || i == 9 {
			dst = append(dst, '-')
		}
	}
	return dst
}

// appendIDText must render ids exactly as the map path's fmt.Sprintf("%v")
// did: strings as-is, otherwise Stringer/%v.
func appendIDText(dst []byte, fv reflect.Value) ([]byte, bool) {
	switch {
	case fv.Kind() == reflect.String:
		return append(dst, fv.String()...), true
	case fv.Type() == uuidType:
		if fv.CanAddr() {
			return appendUUID(dst, fv.Addr().Interface().(*uuid.UUID)), true
		}
		u := fv.Interface().(uuid.UUID)
		return appendUUID(dst, &u), true
	case fv.CanInt():
		return strconv.AppendInt(dst, fv.Int(), 10), true
	case fv.CanUint():
		return strconv.AppendUint(dst, fv.Uint(), 10), true
	case fv.CanInterface():
		if s, ok := fv.Interface().(fmt.Stringer); ok {
			return append(dst, s.String()...), true
		}
		return fmt.Appendf(dst, "%v", fv.Interface()), true
	}
	return nil, false
}

// hasSuffixBytes is strings.HasSuffix against a byte slice. The compiler
// recognizes s[...] == string(suf) and does not copy suf.
func hasSuffixBytes(s string, suf []byte) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == string(suf)
}

// writeJSONString writes a JSON string with the same escaping encoding/json uses
// by default (including HTML-escaping of <, >, & and U+2028/U+2029).
func writeJSONString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	writeJSONStringBody(buf, s)
	buf.WriteByte('"')
}

// writeJSONStringBody omits the surrounding quotes, so callers can build one
// JSON string from several pieces.
func writeJSONStringBody(buf *bytes.Buffer, s string) {
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < 0x80 {
			if htmlSafeSet[b] {
				i++
				continue
			}
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteByte('\\')
			switch b {
			case '\\', '"':
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('t')
			default:
				// Control chars and <, >, & → \u00XX
				buf.WriteString(`u00`)
				buf.WriteByte(hexDigit[b>>4])
				buf.WriteByte(hexDigit[b&0xF])
			}
			i++
			start = i
			continue
		}
		// U+2028 / U+2029 are escaped by encoding/json to keep JSONP-safe output.
		if i+2 < len(s) && s[i] == 0xE2 && s[i+1] == 0x80 && (s[i+2] == 0xA8 || s[i+2] == 0xA9) {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\u202`)
			if s[i+2] == 0xA8 {
				buf.WriteByte('8')
			} else {
				buf.WriteByte('9')
			}
			i += 3
			start = i
			continue
		}
		i++
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
}

// Rendered ids are almost always escape-free (uuid, integer, slug), so only
// the rare case pays for the string conversion.
func writeJSONStringBodyBytes(buf *bytes.Buffer, b []byte) {
	for _, c := range b {
		if c >= 0x80 || !htmlSafeSet[c] {
			writeJSONStringBody(buf, string(b))
			return
		}
	}
	buf.Write(b)
}

const hexDigit = "0123456789abcdef"

// htmlSafeSet marks ASCII bytes that need no escaping under encoding/json's
// default (HTML-escaping) mode. <, >, & and the control range are excluded.
var htmlSafeSet = func() [0x80]bool {
	var s [0x80]bool
	for c := 0x20; c < 0x80; c++ {
		s[c] = true
	}
	s['"'] = false
	s['\\'] = false
	s['<'] = false
	s['>'] = false
	s['&'] = false
	return s
}()
