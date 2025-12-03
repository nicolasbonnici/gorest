# ----------------------------
# Configurable variables
# ----------------------------
DB_TEST_CONTAINER=gorest_db_test

# ----------------------------
# Default target
# ----------------------------
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make codegen         - Run all code generation (models + resources + openapi)"
	@echo "  make codegen-models  - Generate models from database schema"
	@echo "  make codegen-resources - Generate API resources from models"
	@echo "  make codegen-openapi - Generate OpenAPI schema"
	@echo "  make generate        - Alias for codegen"
	@echo "  make test            - Run Go tests"
	@echo "  make test-coverage   - Run Go tests with coverage report"
	@echo "  make benchmark       - Benchmark resource generation (1, 10, 100, 1000 tables)"
	@echo "  make tidy            - Run go mod tidy"

# ----------------------------
# Go targets
# ----------------------------
.PHONY: tidy
tidy:
	@echo "[INFO] Tidying Go modules..."
	@go mod tidy

# ----------------------------
# Code generation targets
# ----------------------------
.PHONY: codegen
codegen:
	@echo "[INFO] Running all code generation..."
	@go run ./cmd/codegen/main.go all

.PHONY: codegen-models
codegen-models:
	@echo "[INFO] Generating models from database schema..."
	@go run ./cmd/codegen/main.go models

.PHONY: codegen-resources
codegen-resources:
	@echo "[INFO] Generating API resources from models..."
	@go run ./cmd/codegen/main.go resources

.PHONY: codegen-openapi
codegen-openapi:
	@echo "[INFO] Generating OpenAPI schema..."
	@go run ./cmd/codegen/main.go openapi

.PHONY: generate
generate: codegen

# ----------------------------
# Test targets
# ----------------------------
.PHONY: test test-up test-schema test-generate
test-up:
	docker compose -f test/compose.yml up -d db_test mysql_test
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
	@export $$(grep -v '^#' test/.env.test | xargs) && $(MAKE) codegen
	@echo "[INFO] Code generation for tests completed"

test: test-up test-schema test-generate
	@echo "[INFO] Running Go tests..."
	@export $$(grep -v '^#' test/.env.test | xargs) && go test -tags=integration -v -timeout=5m ./...
	@echo "[INFO] Restoring auth-enabled resources after tests..."
	@export $$(grep -v '^#' test/.env.test | xargs) && $(MAKE) codegen-resources >/dev/null 2>&1

.PHONY: test-coverage
test-coverage: test-up test-schema test-generate
	@echo "[INFO] Running Go tests with coverage..."
	@mkdir -p coverage
	@export $$(grep -v '^#' test/.env.test | xargs) && go test -tags=integration -timeout=5m -coverprofile=coverage/coverage.out -covermode=atomic ./... 2>&1 | grep -v "go: no such tool"
	@echo ""
	@echo "========================================="
	@echo "         COVERAGE REPORT"
	@echo "========================================="
	@go tool cover -func=coverage/coverage.out | tail -20
	@echo "========================================="
	@go tool cover -func=coverage/coverage.out | grep total | awk '{print "\n📊 Total Coverage: " $$3 "\n"}'
	@echo "[INFO] Restoring auth-enabled resources after tests..."
	@export $$(grep -v '^#' test/.env.test | xargs) && $(MAKE) codegen-resources >/dev/null 2>&1

.PHONY: ci-setup
ci-setup: test-up test-schema
	@echo "[INFO] Generating code for CI..."
	@export $$(grep -v '^#' test/.env.test | xargs) && $(MAKE) codegen
	@echo "[INFO] CI setup complete - database and generated code ready"

# ----------------------------
# Benchmark targets
# ----------------------------
.PHONY: benchmark
benchmark: test-up test-schema
	@export $$(grep -v '^#' test/.env.test | xargs) && go run ./cmd/benchmark/main.go