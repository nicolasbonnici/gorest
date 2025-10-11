//go:build integration

package internal

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"

var db *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	db, err = pgxpool.New(ctx, testDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	code := m.Run()

	db.Close()
	os.Exit(code)
}

// cleanupTestDB truncates all tables to ensure test isolation
func cleanupTestDB(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, "TRUNCATE users, todo CASCADE")
	if err != nil {
		t.Fatalf("Failed to cleanup test database: %v", err)
	}
}
