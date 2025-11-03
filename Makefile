# ----------------------------
# Configurable variables
# ----------------------------
BINARY ?= gorest
DOCKER_IMAGE ?= gorest:latest
DB_CONTAINER ?= gorest_db
API_CONTAINER ?= gorest_api
API_PORT ?= 3000
DB_PORT ?= 5432
DB_URL ?= postgres://postgres:postgres@localhost:$(DB_PORT)/mydb?sslmode=disable
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
	@echo "  make modelgen        - Generate models from database schema"
	@echo "  make resourcegen     - Generate API resources from models (interactive)"
	@echo "  make resourcegen ARGS=-y - Generate resources non-interactively (auto-yes)"
	@echo "  make openapigen      - Generate OpenAPI schema"
	@echo "  make generate        - Run all code generation (models + resources + openapi)"
	@echo "  make docker          - Build and run Docker Compose"
	@echo "  make docker-stop     - Stop Docker Compose"
	@echo "  make docker-clean    - Stop and remove containers/images"
	@echo "  make test            - Run Go tests"
	@echo "  make test-coverage   - Run Go tests with coverage report"
	@echo "  make benchmark       - Benchmark resource generation (1, 10, 100, 1000 tables)"
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
	@go mod tidy

# ----------------------------
# Code generation targets
# ----------------------------
.PHONY: modelgen
modelgen:
	@echo "[INFO] Generating models from database schema..."
	go run ./cmd/modelgen/main.go

.PHONY: resourcegen
resourcegen:
	@echo "[INFO] Generating API resources from models..."
	go run ./cmd/resourcegen/main.go $(ARGS)

.PHONY: openapigen
openapigen:
	@echo "[INFO] Generating OpenAPI schema..."
	go run ./cmd/openapigen/main.go

.PHONY: generate
generate: modelgen
	@echo "[INFO] Generating API resources from models..."
	@go run ./cmd/resourcegen/main.go -y
	@echo "[INFO] Generating OpenAPI schema..."
	@go run ./cmd/openapigen/main.go
	@echo "[INFO] All code generation completed successfully"

# ----------------------------
# Test targets
# ----------------------------
.PHONY: test test-up test-schema test-generate
test-up:
	docker compose -f compose.yml -f compose.override.test.yml up -d db_test mysql_test
	@echo "Waiting for databases to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		docker exec $(DB_TEST_CONTAINER) pg_isready -U postgres >/dev/null 2>&1 && \
		docker exec gorest_mysql_test mysqladmin ping -h 127.0.0.1 -utestuser -ptestpass >/dev/null 2>&1 && \
		echo "✓ Databases ready" && break || \
		(echo "⏳ Waiting for databases... ($$i/10)" && sleep 1); \
		if [ $$i -eq 10 ]; then echo "❌ Timeout waiting for databases"; exit 1; fi; \
	done

test-schema:
	@echo "[INFO] Loading PostgreSQL test schema..."
	docker exec -i $(DB_TEST_CONTAINER) psql -U postgres -d mydb_test < test/sql/schema.sql
	@echo "[INFO] Loading MySQL test schema..."
	docker exec -i gorest_mysql_test mysql -h 127.0.0.1 -utestuser -ptestpass mydb_test < test/sql/schema_mysql.sql

test-generate:
	@echo "[INFO] Code generation for tests..."
	@export $$(grep -v '^#' .env.test | xargs) && $(MAKE) modelgen && $(MAKE) resourcegen ARGS=-y && $(MAKE) openapigen
	@echo "[INFO] Code generation for tests completed"

test: test-up test-schema test-generate
	@echo "[INFO] Running Go tests..."
	@export $$(grep -v '^#' .env.test | xargs) && go test -tags=integration -v -timeout=5m ./...
	@echo "[INFO] Restoring auth-enabled resources after tests..."
	@export $$(grep -v '^#' .env.test | xargs) && $(MAKE) resourcegen ARGS=-y

.PHONY: test-coverage
test-coverage: test-up test-schema test-generate
	@echo "[INFO] Running Go tests with coverage..."
	@mkdir -p coverage
	@export $$(grep -v '^#' .env.test | xargs) && go test -tags=integration -timeout=5m -coverprofile=coverage/coverage.out -covermode=atomic ./... 2>&1 | grep -v "go: no such tool"
	@echo ""
	@echo "========================================="
	@echo "         COVERAGE REPORT"
	@echo "========================================="
	@go tool cover -func=coverage/coverage.out | tail -20
	@echo "========================================="
	@go tool cover -func=coverage/coverage.out | grep total | awk '{print "\n📊 Total Coverage: " $$3 "\n"}'
	@echo "[INFO] Restoring auth-enabled resources after tests..."
	@export $$(grep -v '^#' .env.test | xargs) && $(MAKE) resourcegen ARGS=-y >/dev/null 2>&1

.PHONY: ci-setup
ci-setup: test-up test-schema
	@echo "[INFO] Generating code for CI..."
	@export $$(grep -v '^#' .env.test | xargs) && $(MAKE) modelgen && $(MAKE) resourcegen ARGS=-y && $(MAKE) openapigen
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
	docker compose up --build

.PHONY: docker-stop
docker-stop:
	@echo "[INFO] Stopping Docker Compose..."
	docker compose down

.PHONY: docker-clean
docker-clean:
	@echo "[INFO] Stopping and removing Docker Compose containers and images..."
	docker compose down --rmi all --volumes --remove-orphans

# ----------------------------
# Benchmark targets
# ----------------------------
.PHONY: benchmark
benchmark: test-up test-schema
	@export $$(grep -v '^#' .env.test | xargs) && go run ./cmd/benchmark/main.go

# ----------------------------
# Database targets
# ----------------------------
.PHONY: db-connect
db-connect:
	@echo "[INFO] Connecting to database..."
	psql "$(DB_URL)"