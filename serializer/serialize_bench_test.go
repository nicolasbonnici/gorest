package serializer

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// benchItem mirrors a typical list row: a primary key, a foreign key (exercises
// the IRI-rewrite path), scalars, and timestamps (custom json.Marshaler).
type benchItem struct {
	ID        uuid.UUID `json:"id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	Views     int       `json:"views"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func makeBenchItems(n int) []benchItem {
	items := make([]benchItem, n)
	now := time.Now()
	for i := range items {
		items[i] = benchItem{
			ID:        uuid.New(),
			AuthorID:  uuid.New(),
			Name:      "Benchmark item number that is reasonably long",
			Slug:      "benchmark-item-slug-value",
			Status:    "published",
			Views:     i,
			Active:    true,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return items
}

func benchmarkSerialize(b *testing.B, n int) {
	items := makeBenchItems(n)
	s := &JSONLDSerializer{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := s.Serialize(items, "/benchmarkitems")
		if err != nil {
			b.Fatal(err)
		}
		if len(out) == 0 {
			b.Fatal("empty output")
		}
	}
}

func BenchmarkSerializeJSONLD_10(b *testing.B)   { benchmarkSerialize(b, 10) }
func BenchmarkSerializeJSONLD_100(b *testing.B)  { benchmarkSerialize(b, 100) }
func BenchmarkSerializeJSONLD_1000(b *testing.B) { benchmarkSerialize(b, 1000) }
