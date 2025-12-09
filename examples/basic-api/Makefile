.PHONY: help test-schema generate run build clean

BINARY=bin/api

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable
JWT_SECRET ?= your-secret-key-minimum-32-characters-long

help:
	@echo "Available targets:"
	@echo "  init        - Initialize .env from .env.dist"
	@echo "  db-connect  - Database client connection"
	@echo "  test-schema - Initialize database with test schema and fixtures"
	@echo "  generate    - Generate models, resources, DTOs, and OpenAPI"
	@echo "  run         - Start the API server"
	@echo "  build       - Build the API binary"
	@echo "  clean       - Remove generated code and binary"

init:
	@if [ ! -f .env ]; then \
		cp .env.dist .env && \
		echo "✓ Created .env from .env.dist"; \
		echo "⚠ Please edit .env with your configuration"; \
	else \
		echo "✓ .env already exists"; \
	fi

db-connect:
	@psql $(DATABASE_URL)

test-schema:
	@psql $(DATABASE_URL) -f sql/schema.sql

generate:
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) go run ../../cmd/codegen/main.go all

run:
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) go run .

build:
	@mkdir -p bin
	@VERSION=$$(cd ../.. && git describe --tags --always --dirty 2>/dev/null || echo "dev") && \
	go build -ldflags="-X 'github.com/nicolasbonnici/gorest.Version=$$VERSION'" -o $(BINARY) .

clean:
	@rm -rf generated/
	@rm -rf bin/
