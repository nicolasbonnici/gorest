//go:build integration

package codegen

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

const defaultTestDBURL = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"

var db database.Database

func TestMain(m *testing.M) {
	testDBURL := os.Getenv("DATABASE_URL_TEST")
	if testDBURL == "" {
		testDBURL = os.Getenv("DATABASE_URL")
	}
	if testDBURL == "" {
		testDBURL = defaultTestDBURL
		log.Printf("DATABASE_URL_TEST not set, using default: %s", testDBURL)
	}

	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", testDBURL)
	}

	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-characters-long")
	}

	var err error
	db, err = database.Open("", testDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	log.Printf("✅ Test database connected (%s)", db.DriverName())

	code := m.Run()

	db.Close()
	os.Exit(code)
}

func cleanupTestDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	var err error
	switch db.DriverName() {
	case "postgres":
		_, err = db.Exec(ctx, "TRUNCATE users, todo CASCADE")
	case "mysql":
		_, err = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 0")
		if err == nil {
			_, err = db.Exec(ctx, "TRUNCATE users")
		}
		if err == nil {
			_, err = db.Exec(ctx, "TRUNCATE todo")
		}
		if err == nil {
			_, err = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 1")
		}
	case "sqlite":
		_, err = db.Exec(ctx, "DELETE FROM todo")
		if err == nil {
			_, err = db.Exec(ctx, "DELETE FROM users")
		}
	}

	if err != nil {
		t.Fatalf("Failed to cleanup test database: %v", err)
	}
}
