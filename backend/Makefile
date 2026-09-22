.PHONY: test run up down tidy

test:
	go test ./...

run:
	go run ./cmd/api

tidy:
	go mod tidy

up:
	docker compose up -d

down:
	docker compose down

# Matches docker-compose.yml:
#   export DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
#   export REDIS_URL=redis://localhost:6379/0
.PHONY: test-integration
test-integration:
	go test ./... -count=1 -v -run Integration
