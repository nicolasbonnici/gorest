package crud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

// Same shape as the serializer benchmark's row, so the two numbers compose.
type benchRow struct {
	ID        string    `db:"id"`
	AuthorID  string    `db:"author_id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	Status    string    `db:"status"`
	Views     int64     `db:"views"`
	Active    bool      `db:"active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (benchRow) TableName() string { return "bench_rows" }

const benchSchema = `
CREATE TABLE IF NOT EXISTS bench_rows (
	id TEXT PRIMARY KEY,
	author_id TEXT NOT NULL,
	name TEXT NOT NULL,
	slug TEXT NOT NULL,
	status TEXT NOT NULL,
	views INTEGER NOT NULL,
	active BOOLEAN NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
)`

func setupBenchDB(b *testing.B, rows int) database.Database {
	b.Helper()

	dbURL := fmt.Sprintf("file:%s?mode=memory&cache=shared", b.Name())
	db, err := database.Open("sqlite", dbURL)
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	ctx := context.Background()
	if _, err := db.Exec(ctx, benchSchema); err != nil {
		db.Close()
		b.Fatalf("create schema: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < rows; i++ {
		_, err := db.Exec(ctx,
			`INSERT INTO bench_rows (id, author_id, name, slug, status, views, active, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), uuid.NewString(),
			"Benchmark item number that is reasonably long",
			"benchmark-item-slug-value", "published", int64(i), true, now, now)
		if err != nil {
			db.Close()
			b.Fatalf("seed row %d: %v", i, err)
		}
	}

	b.Cleanup(func() { db.Close() })
	return db
}

func benchmarkGetAllPaginated(b *testing.B, limit int) {
	db := setupBenchDB(b, limit)
	c := New[benchRow](db)
	ctx := context.Background()
	opts := PaginationOptions{Limit: limit, Offset: 0}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := c.GetAllPaginated(ctx, opts)
		if err != nil {
			b.Fatal(err)
		}
		if len(res.Items) != limit {
			b.Fatalf("got %d rows, want %d", len(res.Items), limit)
		}
	}
}

func BenchmarkGetAllPaginated_10(b *testing.B)   { benchmarkGetAllPaginated(b, 10) }
func BenchmarkGetAllPaginated_100(b *testing.B)  { benchmarkGetAllPaginated(b, 100) }
func BenchmarkGetAllPaginated_1000(b *testing.B) { benchmarkGetAllPaginated(b, 1000) }
