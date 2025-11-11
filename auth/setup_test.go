//go:build integration

package auth

import (
	"context"
	"os"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

var db database.Database

func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"
	}

	var err error
	db, err = database.Open("", dsn)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	code := m.Run()

	db.Close()
	os.Exit(code)
}

func cleanupTestDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Cleanup users table
	_, err := db.Exec(ctx, "DELETE FROM users")
	if err != nil {
		t.Logf("Warning: failed to cleanup users table: %v", err)
	}
}
