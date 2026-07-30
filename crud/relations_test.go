package crud

import (
	"context"
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/hooks"
	"github.com/nicolasbonnici/gorest/query"
)

type scopedHooks struct {
	*hooks.NoOpHooks[testModel]
	seenOperations []hooks.Operation
}

func (h *scopedHooks) ModifySelectQuery(ctx context.Context, operation hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
	h.seenOperations = append(h.seenOperations, operation)
	return builder.Where(query.Eq("tenant_id", "tenant-1")), true
}

func rowsForIDs(ids ...int64) database.Rows {
	i := -1
	return &mockRows{
		nextFunc: func() bool {
			i++
			return i < len(ids)
		},
		scanFunc: func(dest ...any) error {
			if id, ok := dest[0].(*int64); ok {
				*id = ids[i]
			}
			if name, ok := dest[1].(*string); ok {
				*name = "Test"
			}
			return nil
		},
	}
}

func TestGetByIDs_AppliesSelectScoping(t *testing.T) {
	var executed string
	var executedArgs []any

	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres"},
		queryFunc: func(ctx context.Context, q string, args ...any) (database.Rows, error) {
			executed = q
			executedArgs = args
			return rowsForIDs(1, 2), nil
		},
	}

	h := &scopedHooks{NoOpHooks: hooks.NewNoOpHooks[testModel]()}
	c := NewWithHooks(db, h)

	if _, err := c.GetByIDs(context.Background(), []any{int64(1), int64(2)}); err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}

	if !strings.Contains(executed, "tenant_id") {
		t.Errorf("scoping from ModifySelectQuery was not applied: %s", executed)
	}

	if !strings.Contains(executed, "IN") {
		t.Errorf("expected a single batched IN query, got: %s", executed)
	}

	var sawTenant bool
	for _, arg := range executedArgs {
		if arg == "tenant-1" {
			sawTenant = true
		}
	}
	if !sawTenant {
		t.Errorf("tenant argument missing from %v", executedArgs)
	}

	if len(h.seenOperations) != 1 || h.seenOperations[0] != hooks.OperationGetByID {
		t.Errorf("expected one GET_BY_ID hook call, got %v", h.seenOperations)
	}
}

func TestGetByIDs_RunsBeforeAndAfterQueryHooks(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres"},
		queryFunc: func(ctx context.Context, q string, args ...any) (database.Rows, error) {
			return rowsForIDs(1), nil
		},
	}

	var before, after bool
	h := newMockHooks()
	h.beforeQueryFunc = func(ctx context.Context, op hooks.Operation, q string, args []any) (string, []any, error) {
		before = op == hooks.OperationGetByID
		return q, args, nil
	}
	h.afterQueryFunc = func(ctx context.Context, op hooks.Operation, q string, args []any, result any, err error) error {
		after = op == hooks.OperationGetByID
		return nil
	}

	c := NewWithHooks(db, h)
	if _, err := c.GetByIDs(context.Background(), []any{int64(1)}); err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}

	if !before {
		t.Error("BeforeQuery was not called for GET_BY_ID")
	}
	if !after {
		t.Error("AfterQuery was not called for GET_BY_ID")
	}
}

func TestRelationFetcher_KeysRowsByID(t *testing.T) {
	var queries int

	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres"},
		queryFunc: func(ctx context.Context, q string, args ...any) (database.Rows, error) {
			queries++
			return rowsForIDs(1, 2, 3), nil
		},
	}

	fetcher := RelationFetcher(New[testModel](db))

	related, err := fetcher.FetchByIDs(context.Background(), []string{"1", "2", "3"})
	if err != nil {
		t.Fatalf("FetchByIDs failed: %v", err)
	}

	if queries != 1 {
		t.Errorf("expected 1 query for 3 ids, got %d", queries)
	}

	for _, id := range []string{"1", "2", "3"} {
		item, ok := related[id]
		if !ok {
			t.Fatalf("id %s missing from the result map", id)
		}
		if _, ok := item.(testModel); !ok {
			t.Errorf("id %s: expected testModel, got %T", id, item)
		}
	}
}

func TestRelationFetcher_NoIDs(t *testing.T) {
	db := &mockDatabase{
		dialect: &mockDialect{name: "postgres"},
		queryFunc: func(ctx context.Context, q string, args ...any) (database.Rows, error) {
			t.Fatal("no query should run for an empty id set")
			return nil, nil
		},
	}

	related, err := RelationFetcher(New[testModel](db)).FetchByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("FetchByIDs failed: %v", err)
	}
	if len(related) != 0 {
		t.Errorf("expected an empty map, got %v", related)
	}
}
