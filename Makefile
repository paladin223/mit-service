# MIT Service Makefile

.PHONY: help build run test clean docker-build docker-run docker-stop deps lint mod-tidy

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application locally"
	@echo "  run-mock      - Run with mock repository"
	@echo "  test          - Run tests"
	@echo "  lint          - Run linter"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  mod-tidy      - Tidy go modules"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose (2 CPU, 2GB RAM)"
	@echo "  docker-stop   - Stop Docker services"
	@echo "  db-up         - Start PostgreSQL only"
	@echo "  db-down       - Stop PostgreSQL"
	@echo "  monitor       - Start performance monitor"
	@echo "  load-test     - Run load test (requires wrk)"

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application locally
run:
	go run ./cmd/server

# Run with mock repository
run-mock:
	REPOSITORY_TYPE=mock go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod verify


# Tidy go modules
mod-tidy:
	go mod tidy

# Build Docker image
docker-build:
	docker build -t mit-service:latest .

# Run with Docker Compose (2 CPU, 2GB RAM limits)
docker-run:
	docker-compose up -d
	@echo "Services started (MIT Service: 2 CPU, 2GB RAM)"
	@echo "Available at http://localhost:8080"

# Run in foreground with logs
docker-run-logs:
	docker-compose up

# Stop Docker services
docker-stop:
	docker-compose down

# Start only PostgreSQL
db-up:
	docker-compose up -d postgres

# Stop PostgreSQL
db-down:
	docker-compose stop postgres
	docker-compose rm -f postgres

# Restart the application service
restart:
	docker-compose restart mit-service

# View logs
logs:
	docker-compose logs -f mit-service

# Setup development environment
dev-setup: deps
	@echo "Development environment setup complete"
	@echo "Run 'make db-up' to start PostgreSQL"
	@echo "Run 'make run' to start the application"

# Full clean (including Docker)
clean-all: clean docker-stop
	docker system prune -f
	docker volume prune -f

# Database migration (if using migrate tool)
migrate-up:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/mitservice?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:password@localhost:5432/mitservice?sslmode=disable" down

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Security check (requires gosec)
security:
	gosec ./...

# Start performance monitor
monitor:
	@chmod +x scripts/monitor.sh
	./scripts/monitor.sh 2

# Run load test
load-test:
	@echo "Starting load test..."
	@echo "Make sure the server is running (make run or make docker-run)"
	@echo "Test will run for 30 seconds with 200 concurrent connections"
	@echo ""
	wrk -t4 -c200 -d30s --script=scripts/load_test_insert.lua http://localhost:8080/insert

# Quick performance check
perf-check:
	@echo "=== Quick Performance Check ==="
	@curl -s http://localhost:8080/performance | jq '.health' || echo "Server not responding"
