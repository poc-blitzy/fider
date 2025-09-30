## This is a self-documented Makefile. For usage information, run `make help`:
##
## For more information, refer to https://www.thapaliya.com/en/writings/well-documented-makefiles/

LDFLAGS += -X github.com/getfider/fider/app/pkg/env.commithash=${COMMITHASH}
LDFLAGS += -X github.com/getfider/fider/app/pkg/env.version=${VERSION}



##@ Running

run: ## Run Fider
	godotenv -f .env ./fider

migrate: ## Run all database migrations
	godotenv -f .env ./fider migrate



##@ Building

build: build-server ## Build server

build-server: ## Build server
	go build -ldflags '-s -w $(LDFLAGS)' -o fider ./cmd



##@ Testing

test: test-server ## Run server tests

test-server: build-server ## Run all server tests
	godotenv -f .test.env ./fider migrate
	godotenv -f .test.env go test ./... -race

coverage-server: build-server ## Run all server tests (with code coverage)
	godotenv -f .test.env ./fider migrate
	godotenv -f .test.env go test ./... -coverprofile=cover.out -coverpkg=all -p=8 -race



##@ E2E Testing

test-e2e-server: ## Run all E2E tests
	npx cucumber-js e2e/features/server/**/*.feature --require-module ts-node/register --require 'e2e/**/*.ts' --publish-quiet



##@ Running (Watch Mode)

watch: watch-server ## Run server in watch mode

watch-server: migrate ## Build and run server in watch mode
	air -c air.conf



##@ Linting

lint: lint-server ## Lint server code

lint-server: ## Lint server code
	golangci-lint run --timeout 3m



##@ Miscellaneous

clean: ## Remove all build-generated content
	rm -rf ./dist
	rm -f ssr.js

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
