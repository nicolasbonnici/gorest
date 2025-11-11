.PHONY: help generate run build clean

BINARY=bin/api
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable
JWT_SECRET ?= your-secret-key-minimum-32-characters-long

help:
	@echo "Available targets:"
	@echo "  generate    - Generate models, resources, DTOs, and OpenAPI"
	@echo "  run         - Start the API server"
	@echo "  build       - Build the API binary"
	@echo "  clean       - Remove generated code and binary"

generate:
	@DATABASE_URL=$(DATABASE_URL) go run github.com/nicolasbonnici/gorest/cmd/modelgen@latest
	@DATABASE_URL=$(DATABASE_URL) go run github.com/nicolasbonnici/gorest/cmd/resourcegen@latest
	@DATABASE_URL=$(DATABASE_URL) go run github.com/nicolasbonnici/gorest/cmd/openapigen@latest

run:
	@DATABASE_URL=$(DATABASE_URL) JWT_SECRET=$(JWT_SECRET) go run .

build:
	@mkdir -p bin
	@go build -o $(BINARY) .

clean:
	@rm -rf generated/
	@rm -rf bin/
