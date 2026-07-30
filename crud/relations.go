package crud

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nicolasbonnici/gorest/internal/idkey"
)

// RelationResolver mirrors serializer.RelationFetcher. Declaring it here rather
// than importing it keeps crud free of a dependency on serializer; the two are
// satisfied structurally.
type RelationResolver interface {
	FetchByIDs(ctx context.Context, ids []string) (map[string]any, error)
}

// RelationFetcher adapts a CRUD instance for serializer.RelationConfig, letting
// relation expansion resolve a whole page with one query per relation instead of
// one per item.
func RelationFetcher[T Model](c *CRUD[T]) RelationResolver {
	return &relationFetcher[T]{crud: c}
}

type relationFetcher[T Model] struct {
	crud *CRUD[T]
}

// FetchByIDs returns the requested rows keyed by identifier. Rows the caller is
// not allowed to read are dropped by the CRUD read hooks and simply stay absent
// from the map.
func (f *relationFetcher[T]) FetchByIDs(ctx context.Context, ids []string) (map[string]any, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var zero T
	meta := getFieldMeta(reflect.TypeOf(zero))
	if meta.idFieldIndex < 0 {
		return nil, fmt.Errorf("cannot expand relation: %s has no id column", zero.TableName())
	}

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	items, err := f.crud.GetByIDs(ctx, args)
	if err != nil {
		return nil, err
	}

	related := make(map[string]any, len(items))
	for i := range items {
		id := reflect.ValueOf(&items[i]).Elem().Field(meta.idFieldIndex).Interface()
		key, ok := idkey.Format(id)
		if !ok {
			continue
		}
		related[key] = items[i]
	}

	return related, nil
}
