GO ?= go
NPM ?= npm
DOCKER ?= docker
STATICCHECK_VERSION ?= 2026.2.2
DEADCODE_VERSION ?= v0.50.0
GO_SOURCES := $(wildcard backend/*.go backend/internal/*/*.go backend/internal/*/*/*.go backend/internal/*/*/*/*.go)

.PHONY: test build run lint deadcode check web docker

test:
	$(if $(GO_SOURCES),,$(error No Go source files exist yet; add a Go package before running tests))
	$(GO) test -race ./...

build:
	$(if $(wildcard backend/main.go),,$(error backend/main.go is missing; the application entry point has not been added yet))
ifeq ($(OS),Windows_NT)
	@if not exist bin mkdir bin
	$(GO) build -o bin/vigil.exe ./backend
else
	@mkdir -p bin
	$(GO) build -o bin/vigil ./backend
endif

run:
	$(if $(wildcard backend/main.go),,$(error backend/main.go is missing; the application entry point has not been added yet))
	$(GO) run ./backend

lint:
	$(if $(GO_SOURCES),,$(error No Go source files exist yet; add a Go package before linting))
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...

deadcode:
	$(if $(wildcard backend/main.go),,$(error backend/main.go is missing; dead-code analysis needs an executable package))
	$(GO) run golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION) -test ./...

check: test build lint deadcode web

web:
	$(if $(wildcard frontend/package.json),,$(error frontend/package.json is missing; the frontend has not been initialized yet))
	$(NPM) --prefix frontend ci
	$(NPM) --prefix frontend run typecheck
	$(NPM) --prefix frontend run build

docker:
	$(if $(wildcard Dockerfile),,$(error Dockerfile is missing; container packaging has not been added yet))
	$(DOCKER) build -t vigil:local .