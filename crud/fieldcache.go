package crud

import (
	"reflect"
	"sync"
)

type fieldMeta struct {
	// SELECT: all db-tagged fields
	allCols    []string
	allIndices []int

	// INSERT: all db-tagged fields except created_at and updated_at
	insertCols    []string
	insertIndices []int

	// UPDATE: all db-tagged fields except id and created_at
	updateCols    []string
	updateIndices []int

	// Struct field index for the "id" column (-1 if absent)
	idFieldIndex int

	// Pools []interface{} slices of len(allIndices) for row scanning
	scanPool sync.Pool
}

var typeCache sync.Map // map[reflect.Type]*fieldMeta

func getFieldMeta(t reflect.Type) *fieldMeta {
	if v, ok := typeCache.Load(t); ok {
		return v.(*fieldMeta)
	}
	meta := buildFieldMeta(t)
	typeCache.Store(t, meta)
	return meta
}

func buildFieldMeta(t reflect.Type) *fieldMeta {
	meta := &fieldMeta{idFieldIndex: -1}

	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		meta.allCols = append(meta.allCols, tag)
		meta.allIndices = append(meta.allIndices, i)

		if tag == "id" {
			meta.idFieldIndex = i
		}

		if tag != "created_at" && tag != "updated_at" {
			meta.insertCols = append(meta.insertCols, tag)
			meta.insertIndices = append(meta.insertIndices, i)
		}

		if tag != "id" && tag != "created_at" {
			meta.updateCols = append(meta.updateCols, tag)
			meta.updateIndices = append(meta.updateIndices, i)
		}
	}

	n := len(meta.allIndices)
	meta.scanPool = sync.Pool{
		New: func() interface{} {
			s := make([]interface{}, n)
			return &s
		},
	}

	return meta
}

func (m *fieldMeta) acquireScanTargets(v reflect.Value) *[]interface{} {
	sp := m.scanPool.Get().(*[]interface{})
	s := *sp
	for i, idx := range m.allIndices {
		s[i] = v.Field(idx).Addr().Interface()
	}
	return sp
}

func (m *fieldMeta) releaseScanTargets(sp *[]interface{}) {
	s := *sp
	for i := range s {
		s[i] = nil
	}
	m.scanPool.Put(sp)
}
