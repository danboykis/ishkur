.PHONY: build clean run

.DEFAULT_GOAL := build

BIN_NAME := ishkur

run: build
	@echo "running $(BIN_NAME)"
	./$(BIN_NAME)

build: clean
	@echo "Building $(BIN_NAME)"
	@go build -o $(BIN_NAME)

clean:
	@rm -f $(BIN_NAME)

test:
	go test ./...

run-local:
	go test -run TestRun
