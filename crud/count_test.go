package crud

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/hooks"
	"github.com/nicolasbonnici/gorest/query"
)

// pageOfRows returns n fully populated testModel rows.
func pageOfRows(n int) database.Rows {
	i := 0
	return &mockRows{
		nextFunc: func() bool {
			i++
			return i <= n
		},
		scanFunc: func(dest ...any) error {
			if id, ok := dest[0].(*int64); ok {
				*id = int64(i)
			}
			for _, d := range dest[1:] {
				if s, ok := d.(*string); ok {
					*s = "value"
				}
			}
			return nil
		},
	}
}

type estimatingDialect struct {
	*mockDialect
	estimateSupported bool
}

func (d *estimatingDialect) EstimateRowsQuery(table string) (string, []any, bool) {
	if !d.estimateSupported {
		return "", nil, false
	}
	return "SELECT reltuples::bigint FROM pg_class WHERE oid = to_regclass($1)", []any{table}, true
}

type filteringHooks struct {
	*hooks.NoOpHooks[testModel]
	denyAll bool
}

func (h *filteringHooks) CheckRead(ctx context.Context, model *testModel) error {
	if h.denyAll {
		return errors.New("forbidden")
	}
	return nil
}

type countProbe struct {
	queries []string
	rows    map[string]int64
	failOn  string
}

func (p *countProbe) queryRow(ctx context.Context, q string, args ...any) database.Row {
	p.queries = append(p.queries, q)
	return &mockRow{
		scanFunc: func(dest ...any) error {
			if p.failOn != "" && strings.Contains(q, p.failOn) {
				return errors.New("query failed")
			}
			for key, value := range p.rows {
				if !strings.Contains(q, key) {
					continue
				}
				switch d := dest[0].(type) {
				case *int64:
					*d = value
				case *int:
					*d = int(value)
				}
			}
			return nil
		},
	}
}

func (p *countProbe) countQueries() int {
	n := 0
	for _, q := range p.queries {
		if strings.Contains(q, "COUNT(*)") {
			n++
		}
	}
	return n
}

func (p *countProbe) estimateQueries() int {
	n := 0
	for _, q := range p.queries {
		if strings.Contains(q, "reltuples") {
			n++
		}
	}
	return n
}

func newCountDB(probe *countProbe, rowsReturned int, estimateSupported bool) *mockDatabase {
	return &mockDatabase{
		dialect: &estimatingDialect{
			mockDialect:       &mockDialect{name: "postgres"},
			estimateSupported: estimateSupported,
		},
		queryFunc: func(ctx context.Context, q string, args ...any) (database.Rows, error) {
			return pageOfRows(rowsReturned), nil
		},
		queryRowFunc: probe.queryRow,
	}
}

func TestCount_ShortPageSkipsCountQuery(t *testing.T) {
	probe := &countProbe{}
	c := New[testModel](newCountDB(probe, 3, false))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		Offset:       20,
		IncludeCount: true,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.countQueries() != 0 {
		t.Errorf("expected no COUNT query for a short page, got %d", probe.countQueries())
	}

	if result.Total == nil {
		t.Fatal("expected an inferred total")
	}
	if *result.Total != 23 {
		t.Errorf("expected total 23 (offset 20 + 3 rows), got %d", *result.Total)
	}
}

func TestCount_FullPageRunsCountQuery(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"COUNT(*)": 512}}
	c := New[testModel](newCountDB(probe, 10, false))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.countQueries() != 1 {
		t.Errorf("expected 1 COUNT query for a full page, got %d", probe.countQueries())
	}
	if result.Total == nil || *result.Total != 512 {
		t.Errorf("expected total 512, got %v", result.Total)
	}
}

func TestCount_UnauthorizedRowsStillCountTowardInference(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"COUNT(*)": 99}}

	h := &filteringHooks{NoOpHooks: hooks.NewNoOpHooks[testModel](), denyAll: true}
	c := NewWithHooks(newCountDB(probe, 4, false), h)

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if len(result.Items) != 0 {
		t.Fatalf("expected every row to be filtered out, got %d", len(result.Items))
	}

	// The database returned 4 rows even though the caller may read none of them.
	// COUNT(*) would report those rows too, so inference must agree.
	if result.Total == nil || *result.Total != 4 {
		t.Errorf("expected total 4 from the rows the database returned, got %v", result.Total)
	}
	if probe.countQueries() != 0 {
		t.Errorf("expected no COUNT query, got %d", probe.countQueries())
	}
}

func TestCount_NoneOmitsTotal(t *testing.T) {
	probe := &countProbe{}
	c := New[testModel](newCountDB(probe, 10, false))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountNone,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if result.Total != nil {
		t.Errorf("expected no total, got %d", *result.Total)
	}
	if probe.countQueries() != 0 {
		t.Errorf("expected no COUNT query, got %d", probe.countQueries())
	}
}

func TestCount_IncludeCountFalseOmitsTotal(t *testing.T) {
	probe := &countProbe{}
	c := New[testModel](newCountDB(probe, 10, false))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:     10,
		CountMode: CountExact,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if result.Total != nil {
		t.Errorf("expected no total, got %d", *result.Total)
	}
	if probe.countQueries() != 0 {
		t.Errorf("expected no COUNT query, got %d", probe.countQueries())
	}
}

func TestCount_EstimateUsesTableStatistics(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"reltuples": 1000000, "COUNT(*)": 42}}
	c := New[testModel](newCountDB(probe, 10, true))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountEstimate,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.countQueries() != 0 {
		t.Errorf("expected the estimate to replace COUNT(*), got %d COUNT queries", probe.countQueries())
	}
	if result.Total == nil || *result.Total != 1000000 {
		t.Errorf("expected the estimated total 1000000, got %v", result.Total)
	}
}

func TestCount_EstimateFallsBackWhenFiltered(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"reltuples": 1000000, "COUNT(*)": 42}}
	c := New[testModel](newCountDB(probe, 10, true))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountEstimate,
		Conditions:   []query.Condition{query.Eq("name", "x")},
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.estimateQueries() != 0 {
		t.Error("a table-wide estimate must not answer a filtered listing")
	}
	if result.Total == nil || *result.Total != 42 {
		t.Errorf("expected the exact total 42, got %v", result.Total)
	}
}

func TestCount_EstimateFallsBackWhenHookScopesQuery(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"reltuples": 1000000, "COUNT(*)": 42}}

	h := &scopedHooks{NoOpHooks: hooks.NewNoOpHooks[testModel]()}
	c := NewWithHooks(newCountDB(probe, 10, true), h)

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountEstimate,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.estimateQueries() != 0 {
		t.Error("a table-wide estimate must not answer a hook-scoped listing")
	}
	if result.Total == nil || *result.Total != 42 {
		t.Errorf("expected the exact total 42, got %v", result.Total)
	}
}

func TestCount_EstimateFallsBackWhenUnsupported(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"COUNT(*)": 42}}
	c := New[testModel](newCountDB(probe, 10, false))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountEstimate,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if result.Total == nil || *result.Total != 42 {
		t.Errorf("expected a fallback to the exact total 42, got %v", result.Total)
	}
}

func TestCount_EstimateFallsBackWhenStatisticsMissing(t *testing.T) {
	// PostgreSQL reports reltuples = -1 for a table it has never analyzed.
	probe := &countProbe{rows: map[string]int64{"reltuples": -1, "COUNT(*)": 42}}
	c := New[testModel](newCountDB(probe, 10, true))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountEstimate,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.countQueries() != 1 {
		t.Errorf("expected a fallback COUNT query, got %d", probe.countQueries())
	}
	if result.Total == nil || *result.Total != 42 {
		t.Errorf("expected the exact total 42, got %v", result.Total)
	}
}

func TestCount_EstimateFallsBackWhenQueryFails(t *testing.T) {
	probe := &countProbe{rows: map[string]int64{"COUNT(*)": 42}, failOn: "reltuples"}
	c := New[testModel](newCountDB(probe, 10, true))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        10,
		IncludeCount: true,
		CountMode:    CountEstimate,
	})
	if err != nil {
		t.Fatalf("a failed estimate must fall back, not fail the request: %v", err)
	}

	if result.Total == nil || *result.Total != 42 {
		t.Errorf("expected the exact total 42, got %v", result.Total)
	}
}

func TestCount_UnlimitedPageInfersTotal(t *testing.T) {
	probe := &countProbe{}
	c := New[testModel](newCountDB(probe, 7, false))

	result, err := c.GetAllPaginated(context.Background(), PaginationOptions{
		Limit:        0,
		IncludeCount: true,
	})
	if err != nil {
		t.Fatalf("GetAllPaginated failed: %v", err)
	}

	if probe.countQueries() != 0 {
		t.Errorf("expected no COUNT query when every row was returned, got %d", probe.countQueries())
	}
	if result.Total == nil || *result.Total != 7 {
		t.Errorf("expected total 7, got %v", result.Total)
	}
}
