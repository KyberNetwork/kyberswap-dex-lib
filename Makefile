lint:
	golangci-lint run ./...

test:
	go test ./... -count=1 -race

.PHONY: lint test
