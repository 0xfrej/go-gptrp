.PHONY: build build-gptrp run-gptrp clean tidy $@

default: clean build

tidy:
	go mod tidy

build: build-gptrp

build-gptrp:
	go build -o ./build/gptrp ./cmd/gptrp

run-gptrp:
	go run ./cmd/gptrp config=./build/config.yaml $(ARGS)

clean:
	rm -rf ./build/gptrp