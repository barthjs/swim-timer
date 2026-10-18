# ==================================================================================== #
# CONFIG
# ==================================================================================== #

APP_NAME	:= swim-timer

GOOS	?= $(shell go env GOOS)
GOARCH	?= $(shell go env GOARCH)
GOARM	?=
OUT		?= bin/$(APP_NAME)

MODULE			:= $(shell go list -m)
# Every Go package except the ones inside web/node_modules, which go list includes.
GO_PKGS			:= $(shell go list ./... | grep -v /node_modules/)
# The feature tests run separately from the unit tests of all other packages.
FEATURE_PKGS	:= $(MODULE)/internal/api
# The build tag of the tests that need an external service.
FEATURE_TAGS	:=
UNIT_PKGS		:= $(filter-out $(FEATURE_PKGS),$(GO_PKGS))
COVER_PKGS		:= $(MODULE)/internal/...
COVER_DIR		:= coverage

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
	docker compose -f compose.dev.yaml up -d
	go run ./cmd/$(APP_NAME)

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## fmt: format all .go files
.PHONY: fmt
fmt:
	@echo 'Formatting .go files...'
	golangci-lint fmt
	swag fmt ./...

## tidy: format all .go files and tidy module dependencies
.PHONY: tidy
tidy: fmt
	@echo 'Tidying module dependencies...'
	go mod tidy
	@echo 'Verifying module dependencies...'
	go mod verify

## test-lint: check formatting and modules without changing files
.PHONY: test-lint
test-lint:
	@echo 'Checking Go formatting...'
	golangci-lint fmt --diff
	@echo 'Checking module dependencies...'
	go mod tidy -diff
	go mod verify

## test-types: run the linters
.PHONY: test-types
test-types:
	@echo 'Linting code...'
	golangci-lint run

## test-unit: run the unit tests with coverage
.PHONY: test-unit
test-unit:
	@echo 'Running unit tests...'
	@mkdir -p $(COVER_DIR)
	@go test -race -vet=off -coverpkg=$(COVER_PKGS) -coverprofile=$(COVER_DIR)/unit.out $(UNIT_PKGS)

## test-feature: run the feature tests with coverage
.PHONY: test-feature
test-feature:
	@echo 'Running feature tests...'
	@mkdir -p $(COVER_DIR)
	go test -race -vet=off -tags='$(FEATURE_TAGS)' -coverpkg=$(COVER_PKGS) -coverprofile=$(COVER_DIR)/feature.out $(FEATURE_PKGS)

## test: run every quality check and print the total coverage
.PHONY: test
test: test-lint test-types test-unit test-feature
	@echo 'Merging coverage profiles...'
	@head -n 1 $(COVER_DIR)/unit.out > $(COVER_DIR)/total.out
	@tail -q -n +2 $(COVER_DIR)/unit.out $(COVER_DIR)/feature.out >> $(COVER_DIR)/total.out
	@go tool cover -func=$(COVER_DIR)/total.out | tail -n 1

## update: update all module dependencies
.PHONY: update
update:
	@echo 'Updating all dependencies...'
	go get -u ./...
	@echo 'Tidying after update...'
	go mod tidy

# ==================================================================================== #
# DOCUMENTATION
# ==================================================================================== #

## docs: generate API documentation
.PHONY: docs
docs: fmt
	swag init -g cmd/$(APP_NAME)/main.go --parseInternal --parseDependency --useStructName --outputTypes go --output docs/api
	@# swag names generic types after the package path of their type arguments, even with --useStructName
	sed -i 's/-internal_api_/-/g' docs/api/docs.go

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build-backend: build the backend
.PHONY: build-backend
build-backend:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags='-s -w' -o=$(OUT) ./cmd/$(APP_NAME)/

## build: build the whole application for the current platform (GOOS/GOARCH/GOARM overridable)
.PHONY: build
build:
	@mkdir -p $(dir $(OUT))
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) goreleaser build --single-target --snapshot --clean -o $(OUT)

## build-linux-386: build for Linux 386
.PHONY: build-linux-386
build-linux-386:
	GOOS=linux GOARCH=386 goreleaser build --single-target --snapshot --clean

## build-linux-amd64: build for Linux AMD64
.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 goreleaser build --single-target --snapshot --clean

## build-linux-armv6: build for Linux ARMv6
.PHONY: build-linux-armv6
build-linux-armv6:
	GOOS=linux GOARCH=arm GOARM=6 goreleaser build --single-target --snapshot --clean

## build-linux-armv7: build for Linux ARMv7
.PHONY: build-linux-armv7
build-linux-armv7:
	GOOS=linux GOARCH=arm GOARM=7 goreleaser build --single-target --snapshot --clean

## build-linux-arm64: build for Linux ARM64
.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 goreleaser build --single-target --snapshot --clean

## build-linux-riscv64: build for Linux RISC-V 64
.PHONY: build-linux-riscv64
build-linux-riscv64:
	GOOS=linux GOARCH=riscv64 goreleaser build --single-target --snapshot --clean

## build-windows-386: build for Windows 386
.PHONY: build-windows-386
build-windows-386:
	GOOS=windows GOARCH=386 goreleaser build --single-target --snapshot --clean

## build-windows-amd64: build for Windows AMD64
.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 goreleaser build --single-target --snapshot --clean

## build-windows-arm64: build for Windows ARM64
.PHONY: build-windows-arm64
build-windows-arm64:
	GOOS=windows GOARCH=arm64 goreleaser build --single-target --snapshot --clean

## build-all: build release archives for all platforms
.PHONY: build-all
build-all:
	goreleaser release --snapshot --clean --skip=publish
