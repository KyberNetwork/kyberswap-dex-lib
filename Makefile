lint:
	golangci-lint run ./...

test:
	go test ./... -count=1

.PHONY: lint test
