# ==================================================================================== #
# CONFIG
# ==================================================================================== #

BINARY_NAME := swim-timer

GOOS    ?= $(shell go env GOOS)
GOARCH  ?= $(shell go env GOARCH)
OUT     ?= bin/$(BINARY_NAME)
LDFLAGS ?= -s -w

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## confirm: ask for confirmation before continuing
.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run the application
.PHONY: run
run:
	go run ./cmd

## update: update all module dependencies
.PHONY: update
update:
	@echo 'Updating all dependencies...'
	go get -u ./...
	@echo 'Tidying after update...'
	go mod tidy

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## fmt: format all .go files
.PHONY: fmt
fmt:
	@echo 'Formatting .go files...'
	go fmt ./...

## tidy: format all .go files and tidy module dependencies
.PHONY: tidy
tidy: fmt
	@echo 'Tidying module dependencies...'
	go mod tidy
	@echo 'Verifying and vendoring module dependencies...'
	go mod verify

## audit: run quality control checks
.PHONY: audit
audit:
	@echo 'Checking module dependencies'
	go mod tidy -diff
	go mod verify
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build-frontend: build only the frontend
.PHONY: build-frontend
build-frontend:
	cd web && \
	npm install && \
	npm run build

## build-backend: build only the backend
.PHONY: build-backend
build-backend:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="$(LDFLAGS)" -o "$(OUT)" ./cmd/

## build: build the whole application
.PHONY: build
build: build-frontend build-backend

## build-linux-x64: build for Linux x64
.PHONY: build-linux-x64
build-linux-x64:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "dist/$(BINARY_NAME)-linux-x64/$(BINARY_NAME)" ./cmd/

## build-linux-arm64: build for Linux ARM64
.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "dist/$(BINARY_NAME)-linux-arm64/$(BINARY_NAME)" ./cmd/

## build-windows-x64: build for Windows x64
.PHONY: build-windows-x64
build-windows-x64:
	go-winres make --in cmd/winres.json --out cmd/rsrc --product-version=git-tag --file-version=git-tag
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS) -H=windowsgui" -o "dist/$(BINARY_NAME)-windows-x64/$(BINARY_NAME).exe" ./cmd/

## build-windows-arm64: build for Windows ARM64
.PHONY: build-windows-arm64
build-windows-arm64:
	go-winres make --in cmd/winres.json --out cmd/rsrc --product-version=git-tag --file-version=git-tag
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS) -H=windowsgui" -o "dist/$(BINARY_NAME)-windows-arm64/$(BINARY_NAME).exe" ./cmd/

## build-all: build for all supported platforms
.PHONY: build-all
build-all: build-frontend build-windows-x64 build-windows-arm64 build-linux-x64 build-linux-arm64

# ==================================================================================== #
# DOCKER
# ==================================================================================== #

## docker-build: build Docker image
.PHONY: docker-build
docker-build:
	docker build -t barthjs/swim-timer:dev --build-arg VERSION=dev .

## docker-run: start services with docker-compose
.PHONY: docker-run
docker-run:
	docker compose up -d

## docker-stop: stop docker-compose services
.PHONY: docker-stop
docker-stop:
	docker compose down
