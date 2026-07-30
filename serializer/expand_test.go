package serializer

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// countingFetcher records how it was called so tests can assert on the number
// of round-trips rather than only on the expanded output.
type countingFetcher struct {
	calls    int
	batches  [][]string
	rows     map[string]interface{}
	failWith error
}

func (f *countingFetcher) FetchByIDs(ctx context.Context, ids []string) (map[string]interface{}, error) {
	f.calls++
	f.batches = append(f.batches, ids)

	if f.failWith != nil {
		return nil, f.failWith
	}

	out := make(map[string]interface{}, len(ids))
	for _, id := range ids {
		if row, ok := f.rows[id]; ok {
			out[id] = row
		}
	}
	return out, nil
}

type expandPost struct {
	ID     string  `json:"id"`
	UserID *string `json:"userId"`
	Title  string  `json:"title"`
}

func makePosts(n int, userIDs []string) []expandPost {
	posts := make([]expandPost, n)
	for i := range posts {
		id := userIDs[i%len(userIDs)]
		posts[i] = expandPost{
			ID:     fmt.Sprintf("post-%d", i),
			UserID: &id,
			Title:  fmt.Sprintf("Post %d", i),
		}
	}
	return posts
}

func userConfig(f *countingFetcher) map[string]RelationConfig {
	return map[string]RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
			RelatedTable:    "users",
			Fetcher:         f,
		},
	}
}

func TestExpandRelations_OneFetchPerRelation(t *testing.T) {
	fetcher := &countingFetcher{rows: map[string]interface{}{
		"user-1": map[string]interface{}{"id": "user-1", "name": "Alice"},
		"user-2": map[string]interface{}{"id": "user-2", "name": "Bob"},
	}}

	posts := makePosts(50, []string{"user-1", "user-2"})

	expanded, err := ExpandRelations(context.Background(), posts, []string{"user"}, userConfig(fetcher))
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	if fetcher.calls != 1 {
		t.Errorf("expected 1 fetch for 50 items, got %d", fetcher.calls)
	}

	if got := len(fetcher.batches[0]); got != 2 {
		t.Errorf("expected the batch to be deduplicated to 2 ids, got %d", got)
	}

	items, ok := expanded.([]interface{})
	if !ok {
		t.Fatalf("expected a slice result, got %T", expanded)
	}
	if len(items) != 50 {
		t.Fatalf("expected 50 items, got %d", len(items))
	}

	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("item %d: expected map, got %T", i, item)
		}
		if _, stillThere := m["userId"]; stillThere {
			t.Errorf("item %d: userId should be replaced by the expanded user", i)
		}
		user, ok := m["user"].(map[string]interface{})
		if !ok {
			t.Fatalf("item %d: expected an expanded user, got %T", i, m["user"])
		}
		wantName := "Alice"
		if i%2 == 1 {
			wantName = "Bob"
		}
		if user["name"] != wantName {
			t.Errorf("item %d: expected user %s, got %v", i, wantName, user["name"])
		}
	}
}

func TestExpandRelations_MultipleRelations(t *testing.T) {
	type article struct {
		ID         string `json:"id"`
		UserID     string `json:"userId"`
		CategoryID string `json:"categoryId"`
	}

	users := &countingFetcher{rows: map[string]interface{}{
		"user-1": map[string]interface{}{"id": "user-1"},
	}}
	categories := &countingFetcher{rows: map[string]interface{}{
		"cat-1": map[string]interface{}{"id": "cat-1"},
	}}

	articles := []article{
		{ID: "a-1", UserID: "user-1", CategoryID: "cat-1"},
		{ID: "a-2", UserID: "user-1", CategoryID: "cat-1"},
	}

	configs := map[string]RelationConfig{
		"user":     {Field: "user", ForeignKeyField: "userId", Fetcher: users},
		"category": {Field: "category", ForeignKeyField: "categoryId", Fetcher: categories},
	}

	expanded, err := ExpandRelations(context.Background(), articles, []string{"user", "category"}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	if users.calls != 1 || categories.calls != 1 {
		t.Errorf("expected 1 fetch per relation, got user=%d category=%d", users.calls, categories.calls)
	}

	items := expanded.([]interface{})
	for i, item := range items {
		m := item.(map[string]interface{})
		if _, ok := m["user"]; !ok {
			t.Errorf("item %d: user not expanded", i)
		}
		if _, ok := m["category"]; !ok {
			t.Errorf("item %d: category not expanded", i)
		}
	}
}

func TestExpandRelations_FetchErrorKeepsForeignKey(t *testing.T) {
	fetcher := &countingFetcher{failWith: errors.New("database unavailable")}

	expanded, err := ExpandRelations(context.Background(), makePosts(3, []string{"user-1"}), []string{"user"}, userConfig(fetcher))
	if err != nil {
		t.Fatalf("a failing relation must not fail the whole response: %v", err)
	}

	for i, item := range expanded.([]interface{}) {
		m := item.(map[string]interface{})
		if _, ok := m["user"]; ok {
			t.Errorf("item %d: user should be absent when the fetch failed", i)
		}
		if _, ok := m["userId"]; !ok {
			t.Errorf("item %d: userId should be preserved when the fetch failed", i)
		}
	}
}

func TestExpandRelations_MissingRowKeepsForeignKey(t *testing.T) {
	fetcher := &countingFetcher{rows: map[string]interface{}{}}

	expanded, err := ExpandRelations(context.Background(), makePosts(1, []string{"ghost"}), []string{"user"}, userConfig(fetcher))
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	m := expanded.([]interface{})[0].(map[string]interface{})
	if _, ok := m["user"]; ok {
		t.Error("user should be absent when the related row does not exist")
	}
	if got := foreignKey(t, m, "userId"); got != "ghost" {
		t.Errorf("userId should be preserved, got %v", got)
	}
}

// foreignKey reads a *string foreign key: structToMap keeps pointer fields as
// pointers instead of flattening them the way a JSON round-trip would.
func foreignKey(t *testing.T, m map[string]interface{}, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("%s not present", key)
	}
	p, ok := v.(*string)
	if !ok || p == nil {
		t.Fatalf("%s: expected a non-nil *string, got %#v", key, v)
	}
	return *p
}

func TestExpandRelations_NoFetcherIsSkipped(t *testing.T) {
	configs := map[string]RelationConfig{
		"user": {Field: "user", ForeignKeyField: "userId"},
	}

	expanded, err := ExpandRelations(context.Background(), makePosts(1, []string{"user-1"}), []string{"user"}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	m := expanded.([]interface{})[0].(map[string]interface{})
	if got := foreignKey(t, m, "userId"); got != "user-1" {
		t.Errorf("userId should be untouched without a fetcher, got %v", got)
	}
}

func TestExpandRelations_NilForeignKeyIsNotFetched(t *testing.T) {
	fetcher := &countingFetcher{rows: map[string]interface{}{}}

	posts := []expandPost{{ID: "post-1", Title: "Orphan"}}

	if _, err := ExpandRelations(context.Background(), posts, []string{"user"}, userConfig(fetcher)); err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	if fetcher.calls != 0 {
		t.Errorf("expected no fetch when every foreign key is nil, got %d", fetcher.calls)
	}
}

func BenchmarkExpandRelations(b *testing.B) {
	rows := make(map[string]interface{}, 25)
	userIDs := make([]string, 25)
	for i := range userIDs {
		id := fmt.Sprintf("user-%d", i)
		userIDs[i] = id
		rows[id] = map[string]interface{}{"id": id, "name": "User"}
	}

	fetcher := &countingFetcher{rows: rows}
	posts := makePosts(25, userIDs)
	configs := userConfig(fetcher)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := ExpandRelations(ctx, posts, []string{"user"}, configs); err != nil {
			b.Fatal(err)
		}
	}
}
