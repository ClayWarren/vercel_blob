.PHONY: fmt lint typecheck test build ci

ci: fmt lint typecheck test build

fmt:
	go fmt ./...

lint:
	golangci-lint run

typecheck:
	go vet ./...

test:
	go test -v ./...

build:
	go build ./...
