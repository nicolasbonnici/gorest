# ----------------------------
# Configurable variables
# ----------------------------
BINARY ?= gorest
DOCKER_IMAGE ?= gorest:latest
DB_CONTAINER ?= gorest_db
API_CONTAINER ?= gorest_api
API_PORT ?= 3000
DB_PORT ?= 5432
DB_URL ?= postgres://postgres:postgres@db:$(DB_PORT)/mydb?sslmode=disable
DB_TEST_SERVICE=db_test
DB_TEST_CONTAINER=gorest_db_test

# ----------------------------
# Default target
# ----------------------------
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make build           - Build the Go binary"
	@echo "  make run             - Run the API locally"
	@echo "  make docker          - Build and run Docker Compose"
	@echo "  make docker-stop     - Stop Docker Compose"
	@echo "  make docker-clean    - Stop and remove containers/images"
	@echo "  make test            - Run Go tests"
	@echo "  make tidy            - Run go mod tidy"
	@echo "  make rebuild         - Clean + build binary"
	@echo ""
	@echo "Configurable variables:"
	@echo "  BINARY=$(BINARY), DOCKER_IMAGE=$(DOCKER_IMAGE), API_PORT=$(API_PORT), DB_PORT=$(DB_PORT)"

# ----------------------------
# Go targets
# ----------------------------
.PHONY: build
build: tidy
	@echo "[INFO] Building Go binary..."
	@mkdir -p bin
	go build -o bin/$(BINARY) ./pkg/gorest

.PHONY: run
run: build
	@echo "[INFO] Running API locally..."
	./bin/$(BINARY)

.PHONY: tidy
tidy:
	@echo "[INFO] Tidying Go modules..."
	@mkdir -p gen/models && echo "package models" > gen/models/.build.go
	@go mod tidy
	@rm -f gen/models/.build.go

.PHONY: test test-up test-schema
test-up:
	docker compose -f compose.yml -f compose.override.test.yml up -d $(DB_TEST_SERVICE)
	@echo "Waiting 2s for DB to be ready..."
	sleep 2

test-schema:
	@echo "[INFO] Loading test database schema..."
	docker exec -i $(DB_TEST_CONTAINER) psql -U postgres -d mydb_test < test/sql/schema.sql

test: test-up test-schema
	go test ./... -v -p=1 -tags=integration

.PHONY: generate
generate:
	@echo "[INFO] Generating models and API from database schema..."
	DATABASE_URL=$(DB_URL) go run ./cmd/genmodels

.PHONY: generate-test
generate-test: test-up test-schema
	@echo "[INFO] Generating models from test database..."
	DATABASE_URL=postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable go run ./cmd/genmodels

.PHONY: ci-setup
ci-setup: test-up test-schema generate-test
	@echo "[INFO] CI setup complete - database and generated code ready"

.PHONY: rebuild
rebuild: clean build

.PHONY: clean
clean:
	@echo "[INFO] Cleaning binary..."
	-rm -f ./bin/$(BINARY)

# ----------------------------
# Docker targets
# ----------------------------
.PHONY: docker
docker:
	@echo "[INFO] Building and running Docker Compose..."
	docker-compose up --build

.PHONY: docker-stop
docker-stop:
	@echo "[INFO] Stopping Docker Compose..."
	docker-compose down

.PHONY: docker-clean
docker-clean:
	@echo "[INFO] Stopping and removing Docker Compose containers and images..."
	docker-compose down --rmi all --volumes --remove-orphans
