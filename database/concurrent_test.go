//go:build integration

package database_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

func setupSQLiteWithSchema(t *testing.T) database.Database {
	t.Helper()
	schema, err := os.ReadFile("../test/sql/schema_sqlite.sql")
	if err != nil {
		t.Fatalf("Failed to read SQLite schema: %v", err)
	}
	return testhelpers.SetupSQLiteWithSchema(t, string(schema))
}

func TestConcurrent_ParallelReadsPostgreSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	testConcurrentParallelReads(t, db)
}

func TestConcurrent_ParallelReadsMySQL(t *testing.T) {
	db := testhelpers.SetupMySQL(t)
	testConcurrentParallelReads(t, db)
}

func TestConcurrent_ParallelReadsSQLite(t *testing.T) {
	db := setupSQLiteWithSchema(t)
	testConcurrentParallelReads(t, db)
}

func testConcurrentParallelReads(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	query := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	for i := 0; i < 10; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		_, err := db.Exec(ctx, query, "User", fmt.Sprintf("%d", i), email)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Ensure MySQL commits data before concurrent reads
	// MySQL may need time to make inserted data visible to other connections
	if db.DriverName() == "mysql" {
		time.Sleep(100 * time.Millisecond)
		// Verify all data is visible
		var count int
		db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if count != 10 {
			t.Fatalf("Expected 10 users after insert, got %d", count)
		}
	}

	var wg sync.WaitGroup
	numReaders := 50
	errChan := make(chan error, numReaders)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			selectQuery := "SELECT COUNT(*) FROM users"
			var count int
			row := db.QueryRow(ctx, selectQuery)
			if err := row.Scan(&count); err != nil {
				errChan <- fmt.Errorf("reader %d failed: %v", id, err)
				return
			}

			if count != 10 {
				errChan <- fmt.Errorf("reader %d got count %d, expected 10", id, count)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

func TestConcurrent_ParallelInsertsPostgreSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	
	testConcurrentParallelInserts(t, db)
}

func TestConcurrent_ParallelInsertsMySQL(t *testing.T) {
	db := testhelpers.SetupMySQL(t)
	
	testConcurrentParallelInserts(t, db)
}

func TestConcurrent_ParallelInsertsSQLite(t *testing.T) {
	t.Skip("SQLite has limited concurrency support - designed for embedded/single-user scenarios")
	db := setupSQLiteWithSchema(t)
	
	testConcurrentParallelInserts(t, db)
}

func testConcurrentParallelInserts(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	var wg sync.WaitGroup
	numWriters := 20
	errChan := make(chan error, numWriters)
	var successCount atomic.Int32

	query := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			email := fmt.Sprintf("concurrent%d@example.com", id)
			_, err := db.Exec(ctx, query, "Concurrent", fmt.Sprintf("User%d", id), email)
			if err != nil {
				errChan <- fmt.Errorf("writer %d failed: %v", id, err)
				return
			}
			successCount.Add(1)
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}

	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count inserted rows: %v", err)
	}

	if count != numWriters {
		t.Errorf("Expected %d rows, got %d", numWriters, count)
	}

	if int(successCount.Load()) != numWriters {
		t.Errorf("Expected %d successful inserts, got %d", numWriters, successCount.Load())
	}
}

func TestConcurrent_ParallelUpdatesPostgreSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	
	testConcurrentParallelUpdates(t, db)
}

func TestConcurrent_ParallelUpdatesMySQL(t *testing.T) {
	db := testhelpers.SetupMySQL(t)
	
	testConcurrentParallelUpdates(t, db)
}

func TestConcurrent_ParallelUpdatesSQLite(t *testing.T) {
	t.Skip("SQLite has limited concurrency support - designed for embedded/single-user scenarios")
	db := setupSQLiteWithSchema(t)
	
	testConcurrentParallelUpdates(t, db)
}

func testConcurrentParallelUpdates(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	for i := 0; i < 10; i++ {
		email := fmt.Sprintf("update%d@example.com", i)
		_, err := db.Exec(ctx, insertQuery, "Original", "Name", email)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	var wg sync.WaitGroup
	numUpdaters := 10
	errChan := make(chan error, numUpdaters)

	updateQuery := "UPDATE users SET firstname = " + db.Dialect().Placeholder(1) +
		" WHERE email = " + db.Dialect().Placeholder(2)

	for i := 0; i < numUpdaters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			email := fmt.Sprintf("update%d@example.com", id)
			newName := fmt.Sprintf("Updated%d", id)
			res, err := db.Exec(ctx, updateQuery, newName, email)
			if err != nil {
				errChan <- fmt.Errorf("updater %d failed: %v", id, err)
				return
			}

			affected, err := res.RowsAffected()
			if err != nil {
				errChan <- fmt.Errorf("updater %d failed to get rows affected: %v", id, err)
				return
			}

			if affected != 1 {
				errChan <- fmt.Errorf("updater %d affected %d rows, expected 1", id, affected)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}

	selectQuery := "SELECT COUNT(*) FROM users WHERE firstname LIKE " + db.Dialect().Placeholder(1)
	var count int
	row := db.QueryRow(ctx, selectQuery, "Updated%")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to verify updates: %v", err)
	}

	if count != numUpdaters {
		t.Errorf("Expected %d updated rows, got %d", numUpdaters, count)
	}
}

func TestConcurrent_MixedOperationsPostgreSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	
	testConcurrentMixedOperations(t, db)
}

func TestConcurrent_MixedOperationsMySQL(t *testing.T) {
	db := testhelpers.SetupMySQL(t)
	
	testConcurrentMixedOperations(t, db)
}

func TestConcurrent_MixedOperationsSQLite(t *testing.T) {
	t.Skip("SQLite has limited concurrency support - designed for embedded/single-user scenarios")
	db := setupSQLiteWithSchema(t)
	
	testConcurrentMixedOperations(t, db)
}

func testConcurrentMixedOperations(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	var wg sync.WaitGroup
	numOperations := 30
	errChan := make(chan error, numOperations)

	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			switch id % 3 {
			case 0:
				email := fmt.Sprintf("mixed%d@example.com", id)
				_, err := db.Exec(ctx, insertQuery, "Mixed", fmt.Sprintf("User%d", id), email)
				if err != nil {
					errChan <- fmt.Errorf("insert %d failed: %v", id, err)
				}

			case 1:
				var count int
				row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users")
				if err := row.Scan(&count); err != nil {
					errChan <- fmt.Errorf("read %d failed: %v", id, err)
				}

			case 2:
				email := fmt.Sprintf("mixed%d@example.com", id-2)
				updateQuery := "UPDATE users SET lastname = " + db.Dialect().Placeholder(1) +
					" WHERE email = " + db.Dialect().Placeholder(2)
				_, err := db.Exec(ctx, updateQuery, "Updated", email)
				if err != nil {
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

func TestConcurrent_TransactionsPostgreSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	
	testConcurrentTransactions(t, db)
}

func TestConcurrent_TransactionsMySQL(t *testing.T) {
	db := testhelpers.SetupMySQL(t)
	
	testConcurrentTransactions(t, db)
}

func TestConcurrent_TransactionsSQLite(t *testing.T) {
	t.Skip("SQLite has limited concurrency support - designed for embedded/single-user scenarios")
	db := setupSQLiteWithSchema(t)
	
	testConcurrentTransactions(t, db)
}

func testConcurrentTransactions(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	var wg sync.WaitGroup
	numTx := 15
	errChan := make(chan error, numTx)
	var commitCount atomic.Int32

	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	for i := 0; i < numTx; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tx, err := db.Begin(ctx)
			if err != nil {
				errChan <- fmt.Errorf("tx %d begin failed: %v", id, err)
				return
			}

			email := fmt.Sprintf("tx%d@example.com", id)
			_, err = tx.Exec(ctx, insertQuery, "Tx", fmt.Sprintf("User%d", id), email)
			if err != nil {
				tx.Rollback(ctx)
				errChan <- fmt.Errorf("tx %d insert failed: %v", id, err)
				return
			}

			if err := tx.Commit(ctx); err != nil {
				errChan <- fmt.Errorf("tx %d commit failed: %v", id, err)
				return
			}

			commitCount.Add(1)
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}

	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count committed rows: %v", err)
	}

	if count != numTx {
		t.Errorf("Expected %d committed transactions, got %d rows", numTx, count)
	}

	if int(commitCount.Load()) != numTx {
		t.Errorf("Expected %d successful commits, got %d", numTx, commitCount.Load())
	}
}

func TestConcurrent_ReadWriteConflictPostgreSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	
	testConcurrentReadWriteConflict(t, db)
}

func TestConcurrent_ReadWriteConflictMySQL(t *testing.T) {
	db := testhelpers.SetupMySQL(t)
	
	testConcurrentReadWriteConflict(t, db)
}

func TestConcurrent_ReadWriteConflictSQLite(t *testing.T) {
	t.Skip("SQLite has limited concurrency support - designed for embedded/single-user scenarios")
	db := setupSQLiteWithSchema(t)
	
	testConcurrentReadWriteConflict(t, db)
}

func testConcurrentReadWriteConflict(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	_, err := db.Exec(ctx, insertQuery, "Conflict", "User", "conflict@example.com")
	if err != nil {
		t.Fatalf("Failed to insert initial data: %v", err)
	}

	var wg sync.WaitGroup
	startChan := make(chan struct{})

	wg.Add(2)

	go func() {
		defer wg.Done()
		<-startChan

		updateQuery := "UPDATE users SET firstname = " + db.Dialect().Placeholder(1) +
			" WHERE email = " + db.Dialect().Placeholder(2)

		for i := 0; i < 50; i++ {
			_, err := db.Exec(ctx, updateQuery, fmt.Sprintf("Writer%d", i), "conflict@example.com")
			if err != nil {
				t.Logf("Writer iteration %d error (acceptable): %v", i, err)
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		<-startChan

		selectQuery := "SELECT firstname FROM users WHERE email = " + db.Dialect().Placeholder(1)

		for i := 0; i < 50; i++ {
			var firstname string
			row := db.QueryRow(ctx, selectQuery, "conflict@example.com")
			if err := row.Scan(&firstname); err != nil {
				t.Errorf("Reader iteration %d failed: %v", i, err)
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	close(startChan)
	wg.Wait()
}

func TestConcurrent_StressTestPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db := testhelpers.SetupPostgresWithDSN(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable&pool_max_conns=10")
	
	testConcurrentStressTest(t, db)
}

func TestConcurrent_StressTestMySQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db := testhelpers.SetupMySQL(t)
	
	testConcurrentStressTest(t, db)
}

func TestConcurrent_StressTestSQLite(t *testing.T) {
	t.Skip("SQLite has limited concurrency support - designed for embedded/single-user scenarios")
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db := setupSQLiteWithSchema(t)
	
	testConcurrentStressTest(t, db)
}

func testConcurrentStressTest(t *testing.T, db database.Database) {
	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 10
	errChan := make(chan error, numGoroutines*operationsPerGoroutine)
	var successCount atomic.Int32

	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				email := fmt.Sprintf("stress%d_%d@example.com", goroutineID, j)

				_, err := db.Exec(ctx, insertQuery, "Stress", fmt.Sprintf("User%d", goroutineID), email)
				if err != nil {
					errChan <- fmt.Errorf("goroutine %d operation %d failed: %v", goroutineID, j, err)
					continue
				}

				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for err := range errChan {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			t.Log("Too many errors, stopping error output...")
			break
		}
	}

	expectedOps := numGoroutines * operationsPerGoroutine
	actualSuccess := int(successCount.Load())

	t.Logf("Stress test: %d/%d operations succeeded", actualSuccess, expectedOps)

	if actualSuccess < expectedOps*95/100 {
		t.Errorf("Success rate too low: %d/%d (%.1f%%)", actualSuccess, expectedOps, float64(actualSuccess)*100/float64(expectedOps))
	}
}
