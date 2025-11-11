.PHONY: help test-schema generate run build clean

BINARY=bin/api
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable
JWT_SECRET ?= your-secret-key-minimum-32-characters-long

help:
	@echo "Available targets:"
	@echo "  db-connect  - Database connection"
	@echo "  test-schema - Initialize database with test schema and fixtures"
	@echo "  generate    - Generate models, resources, DTOs, and OpenAPI"
	@echo "  run         - Start the API server"
	@echo "  build       - Build the API binary"
	@echo "  clean       - Remove generated code and binary"

db-connect:
	@psql $(DATABASE_URL)

test-schema:
	@psql $(DATABASE_URL) -f ../../test/sql/schema.sql

generate:
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) go run ../../cmd/modelgen/main.go
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) yes | go run ../../cmd/resourcegen/main.go
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) go run ../../cmd/openapigen/main.go

run:
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) go run .

build:
	@mkdir -p bin
	@go build -o $(BINARY) .

clean:
	@rm -rf generated/
	@rm -rf bin/
