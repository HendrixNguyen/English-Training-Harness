.PHONY: test run up down tidy

test:
	go test ./...

# The Go process does not read .env. `make run` sources it here, in this
# recipe's own shell (make runs each recipe line in a fresh /bin/sh), so
# nothing from .env — least of all a TEST_* variable, which would un-gate the
# destructive integration tests — leaks into your interactive shell.
#   cp .env.example .env   # once; then edit, then:
#   make run
run:
	@test -f .env || { echo ".env missing: cp .env.example .env, then edit it"; exit 1; }
	set -a; . ./.env; set +a; exec go run ./cmd/api

tidy:
	go mod tidy

# What CI's backend-unit job runs, so a branch can be checked before it is
# pushed — including its first step: the unit suite must prove it needs no
# service (an exported TEST_* would also un-gate the destructive integration
# tests and run them without -p 1).
.PHONY: no-service-vars build fmt-check vet check
no-service-vars:
	@for v in DATABASE_URL REDIS_URL TEST_DATABASE_URL TEST_REDIS_URL; do \
	  if [ -n "$$(printenv "$$v")" ]; then \
	    echo "$$v is set: make check mirrors CI's backend-unit, which runs with no service variables (unset $$v, or run it in a subshell)"; exit 1; \
	  fi; \
	done

build:
	go build ./...

# gofmt -l exits 2 on a file it cannot parse and lists nothing; fail on that
# too, or a syntax error would pass this check (the message is already on stderr).
fmt-check:
	@unformatted=$$(gofmt -l .) || { echo "gofmt -l failed (see above)"; exit 1; }; \
	if [ -n "$$unformatted" ]; then echo "gofmt -l:"; echo "$$unformatted"; exit 1; fi

vet:
	go vet ./...

check: no-service-vars build fmt-check vet
	go test ./... -count=1 -race

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
# -p 1: every package's integration tests share one physical database, and
# internal/store destructively drops/recreates the schema around its own
# tests, so running packages' test binaries in parallel (go test's default)
# races. See .github/workflows/ci.yml's backend-integration job.
# -race: three of these tests drive shared code from eight goroutines (pet
# Ensure and PenaliseMiss, store Migrate) — the only place the real
# repositories are exercised concurrently.
.PHONY: test-integration
test-integration:
	go test ./... -count=1 -v -run Integration -p 1 -race
