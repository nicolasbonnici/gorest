# ----------------------------
# Configurable variables
# ----------------------------
DB_TEST_CONTAINER=gorest_db_test

# Version from git tag, fallback to git describe, or "dev" if no git
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build flags for version injection
LDFLAGS=-ldflags "-X 'github.com/nicolasbonnici/gorest.Version=$(VERSION)'"

# ----------------------------
# Default target
# ----------------------------
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make version         - Show current version"
	@echo "  make lint            - Run golangci-lint to check code"
	@echo "  make lint-fix        - Run golangci-lint with --fix for auto-fixable issues"
	@echo "  make test            - Run Go tests"
	@echo "  make test-coverage   - Run Go tests with coverage report"
	@echo "  make tidy            - Run go mod tidy"
	@echo ""
	@echo "Note: Code generation is now in the gorest-codegen plugin"
	@echo "      See: https://github.com/nicolasbonnici/gorest-codegen"

# ----------------------------
# Go targets
# ----------------------------
.PHONY: tidy
tidy:
	@echo "[INFO] Tidying Go modules..."
	@go mod tidy

# ----------------------------
# Version target
# ----------------------------
.PHONY: version
version:
	@echo "$(VERSION)"

# ----------------------------
# Linting targets
# ----------------------------
.PHONY: lint
lint:
	@echo "[INFO] Running go vet (excluding benchmark testserver)..."
	@packages=$$(go list ./... 2>/dev/null | grep -v '/plugins/benchmark/testserver' || true); \
	if [ -n "$$packages" ]; then \
		go vet $$packages; \
	fi
	@echo "[INFO] Checking formatting..."
	@unformatted=$$(gofmt -l . | grep -v '^vendor/' | grep -v 'generated/' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files need formatting:"; \
		echo "$$unformatted"; \
		echo "Run 'make lint-fix' to fix"; \
		exit 1; \
	fi
	@echo "[INFO] All checks passed!"

.PHONY: lint-fix
lint-fix:
	@echo "[INFO] Fixing formatting issues..."
	@gofmt -w -s $$(find . -name '*.go' | grep -v vendor | grep -v /generated/)
	@echo "[INFO] Formatting fixed!"

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
	@if [ ! -f ../go/gorest-codegen/gorest-codegen ]; then \
		echo "[INFO] Building gorest-codegen binary..."; \
		(cd ../go/gorest-codegen && go build -o gorest-codegen ./cmd/codegen); \
	fi
	@export $$(grep -v '^#' test/.env.test | xargs) && \
		(cd test && ../../go/gorest-codegen/gorest-codegen all)
	@echo "[INFO] Code generation for tests completed"

test: test-up test-schema test-generate
	@echo "[INFO] Running Go tests..."
	@export $$(grep -v '^#' test/.env.test | xargs) && go test -p 1 -tags=integration -v -timeout=5m ./...
	@echo "[INFO] Restoring auth-enabled resources after tests..."
	@export $$(grep -v '^#' test/.env.test | xargs) && \
		(cd test && ../../go/gorest-codegen/gorest-codegen resources >/dev/null 2>&1)

.PHONY: test-coverage
test-coverage: test-up test-schema test-generate
	@echo "[INFO] Running Go tests with coverage..."
	@mkdir -p coverage
	@export $$(grep -v '^#' test/.env.test | xargs) && \
		packages=$$(go list ./... | grep -v 'github.com/nicolasbonnici/gorest$$' | grep -v '/cmd/' | grep -v '/examples/' | grep -v '/plugins/' | grep -v '/generated/' | grep -v '/test/generated/' | grep -v '/hooks$$' | grep -v '/logger$$' | grep -v '/migrations/cmd/') && \
		go test -p 1 -tags=integration -timeout=5m -coverprofile=coverage/coverage.out -covermode=atomic $$packages
	@echo ""
	@echo "========================================="
	@echo "         COVERAGE REPORT"
	@echo "========================================="
	@go tool cover -func=coverage/coverage.out | column -t
	@echo "========================================="
	@go tool cover -func=coverage/coverage.out | grep total | awk '{print "\n📊 Total Coverage: " $$3 "\n"}'
	@echo "[INFO] Restoring auth-enabled resources after tests..."
	@export $$(grep -v '^#' test/.env.test | xargs) && \
		(cd test && ../../go/gorest-codegen/gorest-codegen resources >/dev/null 2>&1)

.PHONY: ci-setup
ci-setup: test-up test-schema
	@echo "[INFO] Generating code for CI..."
	@if [ ! -f ../go/gorest-codegen/gorest-codegen ]; then \
		echo "[INFO] Building gorest-codegen binary..."; \
		(cd ../go/gorest-codegen && go build -o gorest-codegen ./cmd/codegen); \
	fi
	@export $$(grep -v '^#' test/.env.test | xargs) && \
		(cd test && ../../go/gorest-codegen/gorest-codegen all)
	@echo "[INFO] CI setup complete - database and generated code ready"

# ----------------------------
# TODO move on benchmarkmark plugin makefile
# Benchmark targets
# ----------------------------
.PHONY: benchmark
benchmark: test-up test-schema
	@export $$(grep -v '^#' test/.env.test | xargs) && go run ./plugins/benchmark/cmd/main.go