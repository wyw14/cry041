.PHONY: test test-race vet build web migrate seed up down
test:
	go test ./...
test-race:
	go test -race ./...
vet:
	go vet ./...
build:
	go build ./...
web:
	cd web && npm ci && npm test && npm run build
migrate:
	psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/001_init.sql
seed:
	psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/002_seed.sql
up:
	docker compose up --build
down:
	docker compose down
