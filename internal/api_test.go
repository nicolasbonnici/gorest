package internal_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicolasbonnici/gorest/internal"
)

var tables = map[string]internal.TableSchema{
	"users": {
		TableName: "users",
		Columns:   []string{"id", "name", "email"},
	},
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	connStr := "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connStr)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name TEXT,
			email TEXT
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `TRUNCATE TABLE users RESTART IDENTITY`)
	require.NoError(t, err)

	return pool
}

func setupTestApp(db *pgxpool.Pool) *fiber.App {
	app := fiber.New()
	internal.SetupAPI(app, db, tables, "secret")
	return app
}

func TestGetUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(context.Background(), `INSERT INTO users(name,email) VALUES('Alice','alice@example.com')`)
	require.NoError(t, err)

	app := setupTestApp(db)

	req := httptest.NewRequest(http.MethodGet, "/users?limit=1", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPostUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	app := setupTestApp(db)

	body := []byte(`{"name":"Bob","email":"bob@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var count int
	err = db.QueryRow(context.Background(), `SELECT COUNT(*) FROM users WHERE name='Bob'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
