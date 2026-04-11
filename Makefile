# ==================================================================================== #
# CONFIG
# ==================================================================================== #

BINARY_NAME := swim-timer
VERSION     ?= dev

GOOS    ?= $(shell go env GOOS)
GOARCH  ?= $(shell go env GOARCH)
OUT     ?= bin/$(BINARY_NAME)
LDFLAGS ?= -s -w -X 'main.Version=$(VERSION)'

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
	export VITE_APP_VERSION=$(VERSION) && \
	npm install && \
	npm run build

## build-backend: build only the backend
.PHONY: build-backend
build-backend:
	GOOS=$(GOOS) GOARCH=$(GOARCH) VERSION=$(VERSION) OUT=$(OUT) sh ./scripts/go-build.sh

## build: build the whole application
.PHONY: build
build: build-frontend build-backend

## build-linux-386: build for Linux 386
.PHONY: build-linux-386
build-linux-386: build-frontend
	GOOS=linux GOARCH=386 VERSION=$(VERSION) OUT=dist/$(BINARY_NAME)_linux_386/$(BINARY_NAME) sh ./scripts/go-build.sh

## build-linux-amd64: build for Linux AMD64
.PHONY: build-linux-amd64
build-linux-amd64: build-frontend
	GOOS=linux GOARCH=amd64 VERSION=$(VERSION) OUT=dist/$(BINARY_NAME)_linux_amd64/$(BINARY_NAME) sh ./scripts/go-build.sh

## build-linux-arm64: build for Linux ARM64
.PHONY: build-linux-arm64
build-linux-arm64: build-frontend
	GOOS=linux GOARCH=arm64 VERSION=$(VERSION) OUT=dist/$(BINARY_NAME)_linux_arm64/$(BINARY_NAME) sh ./scripts/go-build.sh

## build-winres: create Windows resources
.PHONY: build-winres
build-winres:
	go-winres make --in cmd/winres.json --out cmd/rsrc --product-version=git-tag --file-version=git-tag

## build-windows-386: build for Windows 386
.PHONY: build-windows-386
build-windows-386: build-frontend build-winres
	GOOS=windows GOARCH=386 VERSION=$(VERSION) OUT=dist/$(BINARY_NAME)_windows_386/$(BINARY_NAME).exe sh ./scripts/go-build.sh

## build-windows-amd64: build for Windows AMD64
.PHONY: build-windows-amd64
build-windows-amd64: build-frontend build-winres
	GOOS=windows GOARCH=amd64 VERSION=$(VERSION) OUT=dist/$(BINARY_NAME)_windows_amd64/$(BINARY_NAME).exe sh ./scripts/go-build.sh

## build-windows-arm64: build for Windows ARM64
.PHONY: build-windows-arm64
build-windows-arm64: build-frontend build-winres
	GOOS=windows GOARCH=arm64 VERSION=$(VERSION) OUT=dist/$(BINARY_NAME)_windows_arm64/$(BINARY_NAME).exe sh ./scripts/go-build.sh

## build-all: build for all supported platforms
.PHONY: build-all
build-all:
	./scripts/build-release.sh $(VERSION)

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
