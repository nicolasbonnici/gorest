package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal"
)

func main() {
	dbURL := "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("❌ DB connection failed: %v", err)
	}
	defer db.Close()

	tables := internal.LoadSchema(db)
	internal.GenerateStructs(tables)
	internal.GenerateCRUD()
	internal.GenerateAPI(db, tables)

	log.Println("✅ Code generation completed successfully")
}
