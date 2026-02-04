.PHONY: dev-backend dev-frontend build test clean

# Development
dev-backend:
	go run ./cmd/server

dev-frontend:
	cd web && npm run dev

# Build
build: build-frontend build-backend

build-frontend:
	cd web && npm ci && npm run build

build-backend:
	go build -o bin/server ./cmd/server

# Test
test: test-backend test-frontend

test-backend:
	go test ./...

test-frontend:
	cd web && npm test -- --run

# Lint
lint-backend:
	go vet ./...

lint-frontend:
	cd web && npm run lint

# Clean
clean:
	rm -rf bin/ web/dist/

# Docker
docker-build:
	docker build -t flyagi .

docker-run:
	docker run -p 8080:8080 flyagi
