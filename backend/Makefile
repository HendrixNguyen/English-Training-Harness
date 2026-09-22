.PHONY: test run up down tidy

test:
	go test ./...

run:
	go run ./cmd/api

tidy:
	go mod tidy

# Ports come from .env (copy .env.example). Set REDIS_PORT / POSTGRES_PORT there
# if 6379 or 5432 is already taken on this machine; `down` keeps the postgres
# volume, `docker compose down -v` discards it.
up:
	docker compose up -d

down:
	docker compose down

# Live-database tests. They DROP every table, so they are gated on test-only
# variables and never on the production DATABASE_URL / REDIS_URL of spec §8.
# Copy .env.example, then:
#   export TEST_DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
#   export TEST_REDIS_URL=redis://localhost:6379/0
# Without them every Integration test skips.
.PHONY: test-integration
test-integration:
	go test ./... -count=1 -v -run Integration
