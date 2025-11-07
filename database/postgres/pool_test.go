//go:build integration

package postgres

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

const testPoolDSN = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable&pool_max_conns=5&pool_min_conns=1"

func TestPool_ConnectionReuse(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		var result int
		row := db.QueryRow(ctx, "SELECT 1")
		if err := row.Scan(&result); err != nil {
			t.Fatalf("Query %d failed: %v", i, err)
		}
	}
}

func TestPool_ConcurrentQueries(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 20
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			var result int
			row := db.QueryRow(ctx, "SELECT $1::integer", id)
			if err := row.Scan(&result); err != nil {
				errChan <- err
				return
			}

			if result != id {
				errChan <- fmt.Errorf("expected %d, got %d", id, result)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent query error: %v", err)
	}
}

func TestPool_MaxConnections(t *testing.T) {
	limitedDSN := "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable&pool_max_conns=2"
	db, err := database.Open("postgres", limitedDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	startChan := make(chan struct{})

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startChan

			var result int
			row := db.QueryRow(ctx, "SELECT pg_sleep(0.1), 1")
			row.Scan(&result)
		}()
	}

	close(startChan)
	wg.Wait()
}

func TestPool_ConnectionTimeout(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var result int
	row := db.QueryRow(ctx, "SELECT pg_sleep(10)")
	err = row.Scan(&result)
	if err == nil {
		t.Error("Expected timeout error, got none")
	}

	if err != nil && !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "cancel") {
		t.Logf("Got error (expected context timeout): %v", err)
	}
}

func TestPool_Ping(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := db.Ping(ctx); err != nil {
			t.Fatalf("Ping %d failed: %v", i, err)
		}
	}
}

func TestPool_AfterClose(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	db.Close()

	ctx := context.Background()
	_, err = db.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error after close, got none")
	}
}

func TestPool_TransactionPooling(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	numTransactions := 10

	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tx, err := db.Begin(ctx)
			if err != nil {
				t.Errorf("Failed to begin transaction %d: %v", id, err)
				return
			}

			var result int
			row := tx.QueryRow(ctx, "SELECT $1::integer", id)
			if err := row.Scan(&result); err != nil {
				tx.Rollback(ctx)
				t.Errorf("Query failed in transaction %d: %v", id, err)
				return
			}

			if err := tx.Commit(ctx); err != nil {
				t.Errorf("Failed to commit transaction %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}

func TestPool_LongRunningQuery(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var result int
		row := db.QueryRow(ctx, "SELECT pg_sleep(1), 1")
		row.Scan(&result)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)

		var result int
		row := db.QueryRow(ctx, "SELECT 2")
		if err := row.Scan(&result); err != nil {
			t.Errorf("Short query failed while long query running: %v", err)
		}
	}()

	wg.Wait()
}

func TestPool_RowsNotClosed(t *testing.T) {
	db, err := database.Open("postgres", testPoolDSN)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		rows, err := db.Query(ctx, "SELECT generate_series(1, 100)")
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		var count int
		for rows.Next() && count < 5 {
			var n int
			rows.Scan(&n)
			count++
		}
		rows.Close()
	}

	var result int
	row := db.QueryRow(ctx, "SELECT 1")
	if err := row.Scan(&result); err != nil {
		t.Errorf("Pool exhausted after unclosed rows: %v", err)
	}
}
