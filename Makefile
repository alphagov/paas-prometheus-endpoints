.PHONY: build clean test

APP_ROOT ?= $(PWD)

build: bin/example bin/redis

bin/example: clean
	go build -mod=vendor -o ./bin/example ./src/example

bin/redis: clean
	go build -mod=vendor -o ./bin/redis ./src/redis

clean:
	rm -f bin/redis

test:
	go test -mod=vendor $$(go list github.com/alphagov/paas-prometheus-endpoints/pkg/...)
	go test -mod=vendor ./...
