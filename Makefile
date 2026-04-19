# ============================================================
# Makefile — shortcut commands untuk development
# ============================================================
.PHONY: run build docker-build docker-run migrate-up migrate-down tidy

APP_NAME=go-restapi
IMAGE_NAME=go-restapi:latest

## run: jalankan aplikasi secara lokal (development)
run:
	go run ./cmd/api/...

## build: compile binary lokal
build:
	CGO_ENABLED=0 go build -o bin/$(APP_NAME) ./cmd/api

## tidy: bersihkan dan sinkronkan go.mod
tidy:
	go mod tidy

## docker-build: build Docker image (multi-stage)
docker-build:
	docker build -t $(IMAGE_NAME) .
	@echo "Image size:"
	@docker image inspect $(IMAGE_NAME) --format='{{.Size}}' | awk '{printf "%.2f MB\n", $$1/1024/1024}'

## docker-run: jalankan container dengan .env file
docker-run:
	docker run --rm -p 8088:8088 \
		--env-file .env \
		--add-host host.docker.internal:host-gateway \
		--name $(APP_NAME) \
		$(IMAGE_NAME)

## compose-up: jalankan via docker compose
compose-up:
	docker compose up --build -d

## compose-down: stop dan hapus container
compose-down:
	docker compose down

## migrate-up: jalankan migrasi database
migrate-up:
	./scripts/migrate.sh up

## migrate-down: rollback migrasi terakhir
migrate-down:
	./scripts/migrate.sh down

## lint: jalankan golangci-lint
lint:
	golangci-lint run ./...

## test: jalankan semua unit test
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
