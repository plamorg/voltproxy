BINARY_NAME=voltproxy

default: build

build:
	go build -o $(BINARY_NAME) -v

run:
	go run ./...

test:
	go test -count=1 ./...

lint:
	golangci-lint run

clean:
	go clean
	rm -f $(BINARY_NAME)

.PHONY: build run test clean
