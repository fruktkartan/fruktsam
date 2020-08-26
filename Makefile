all: lint

lint:
	golangci-lint run
