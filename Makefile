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
	go build -o $(BINARY) ./cmd/server.go

.PHONY: run
run: build
	@echo "[INFO] Running API locally..."
	./$(BINARY)

.PHONY: tidy
tidy:
	@echo "[INFO] Tidying Go modules..."
	go mod tidy

.PHONY: test
test:
	@echo "[INFO] Running tests..."
	go test ./...

.PHONY: rebuild
rebuild: clean build

.PHONY: clean
clean:
	@echo "[INFO] Cleaning binary..."
	-rm -f $(BINARY)

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
