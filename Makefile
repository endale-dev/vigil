GO ?= go
NPM ?= npm
DOCKER ?= docker
STATICCHECK_VERSION ?= 2026.2.2
DEADCODE_VERSION ?= v0.50.0
CGO_ENABLED ?= 1
export CGO_ENABLED
GO_SOURCES := $(wildcard backend/*.go backend/internal/*/*.go backend/internal/*/*/*.go backend/internal/*/*/*/*.go)

.PHONY: backend-module test build run lint deadcode check web docker
.NOTPARALLEL: check

backend-module:
ifeq ($(wildcard backend/go.mod),)
	$(GO) -C backend mod init github.com/endale-dev/vigil
endif
	$(GO) -C backend mod edit -go=1.26
	$(GO) -C backend get go.mongodb.org/mongo-driver/v2@v2.10.1
	$(GO) -C backend mod tidy

test: backend-module
	$(if $(GO_SOURCES),,$(error No Go source files exist yet; add a Go package before running tests))
	$(GO) -C backend test -race ./...

build: backend-module web
	$(if $(wildcard backend/main.go),,$(error backend/main.go is missing; the application entry point has not been added yet))
ifeq ($(OS),Windows_NT)
	@if not exist backend\bin mkdir backend\bin
	$(GO) -C backend build -o bin/vigil.exe .
else
	@mkdir -p backend/bin
	$(GO) -C backend build -o bin/vigil .
endif

run: backend-module
	$(if $(wildcard backend/main.go),,$(error backend/main.go is missing; the application entry point has not been added yet))
	$(GO) -C backend run .

lint: backend-module
	$(if $(GO_SOURCES),,$(error No Go source files exist yet; add a Go package before linting))
	$(GO) -C backend vet ./...
	$(GO) -C backend run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...

deadcode: backend-module
	$(if $(wildcard backend/main.go),,$(error backend/main.go is missing; dead-code analysis needs an executable package))
	$(GO) -C backend run golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION) -test ./...

check: test web build lint deadcode

web:
	$(if $(wildcard frontend/package.json),,$(error frontend/package.json is missing; the frontend has not been initialized yet))
	$(if $(wildcard frontend/node_modules/vite/package.json),,$(NPM) --prefix frontend ci)
	$(NPM) --prefix frontend run typecheck
	$(NPM) --prefix frontend run build
	node -e "require('node:fs').cpSync('frontend/dist', 'backend/dist', { recursive: true })"

docker:
	$(if $(wildcard backend/Dockerfile),,$(error backend/Dockerfile is missing; container packaging has not been added yet))
	$(DOCKER) build -f backend/Dockerfile -t vigil:local .
