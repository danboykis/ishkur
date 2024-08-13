.PHONY: build clean run

.DEFAULT: build

run: build
	@echo "running ishkur"
	./ishkur

build: clean
	@echo "Building ishkur"
	@go build -o ishkur

clean:
	@rm -f ishkur

test:
	go test ./...

run-local:
	go test -run TestRun
