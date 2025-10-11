//go:build integration

package internal

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"

var db *pgxpool.Pool

func TestMain(m *testing.M) {
	var err error
	db, err = pgxpool.New(context.Background(), testDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	code := m.Run()

	db.Close()
	os.Exit(code)
}
