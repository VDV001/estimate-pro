# EstimatePro — development commands

set dotenv-load

# Dev infrastructure (postgres, redis, minio)
dev-infra:
    docker compose up -d

# Stop infrastructure
stop-infra:
    docker compose down

# Run Go backend
dev-backend:
    cd backend && go1.26.0 run ./cmd/server

# Run Next.js frontend
dev-frontend:
    cd frontend && npm run dev

# Database migrations
migrate-up:
    cd backend && goose -dir migrations postgres "$DATABASE_URL" up

migrate-down:
    cd backend && goose -dir migrations postgres "$DATABASE_URL" down

migrate-create name:
    cd backend && goose -dir migrations create {{name}} sql

migrate-status:
    cd backend && goose -dir migrations postgres "$DATABASE_URL" status

# Tests
test-backend:
    cd backend && go1.26.0 test ./...

test-backend-v:
    cd backend && go1.26.0 test -v ./...

# Build
build-backend:
    cd backend && go1.26.0 build -o bin/server ./cmd/server

# Lint
lint-backend:
    cd backend && golangci-lint run ./...
