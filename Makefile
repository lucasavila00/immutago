GOMARKDOC = go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0
GOLANGCI = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
DOC_PACKAGES = ./cmd/immutago ./pkg/analyzer ./pkg/immutable ./internal/checker

.PHONY: build test test-unit test-e2e docs docs-check lint

build:
	go build -o bin/immutago ./cmd/immutago

test:
	go test -race ./...

test-unit:
	go test ./internal/... ./pkg/... ./scripts/...

test-e2e:
	go test ./tests -count=1

docs:
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) "$$package"; done

docs-check:
	@set -e; for package in $(DOC_PACKAGES); do $(GOMARKDOC) --check "$$package"; done

lint:
	go run ./scripts/check-source-size.go
	go run ./scripts/check-function-comments
	sh scripts/check-dead-code.sh
	$(GOLANGCI) run ./...
	$(GOLANGCI) fmt --diff ./...
