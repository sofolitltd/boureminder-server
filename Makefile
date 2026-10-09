.PHONY: dev build build-linux deploy test lint migrate-up migrate-down river-migrate sqlc

dev:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

# Cross-compile for the Coolify VPS (linux/amd64). Produces
# ./boureminder-api, which the Dockerfile just COPYs into an Alpine
# image — no build step runs on the server itself.
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o boureminder-api ./cmd/server

# Build the deploy binary, commit it, and push — see coolify_deploy.md.
deploy: build-linux
	git add boureminder-api
	git commit -m "build: deploy"
	git push origin main

test:
	go test ./...

lint:
	gofmt -l .
	go vet ./...

# Schema migrations (goose), run against the direct (non-pooled) connection.
migrate-up:
	go tool goose -dir internal/db/migrations postgres "$$DATABASE_DIRECT_URL" up

migrate-down:
	go tool goose -dir internal/db/migrations postgres "$$DATABASE_DIRECT_URL" down

# River's own job-queue tables, same direct connection.
river-migrate:
	go tool river migrate-up --database-url "$$DATABASE_DIRECT_URL"

sqlc:
	go tool sqlc generate
