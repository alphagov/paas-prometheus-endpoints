.PHONY: build clean test

APP_ROOT ?= $(PWD)

build: bin/example bin/redis bin/elasticsearch

bin/example: clean
	go build -mod=vendor -o ./bin/example ./src/example

bin/redis: clean
	go build -mod=vendor -o ./bin/redis ./src/redis

bin/elasticsearch: clean
	go build -mod=vendor -o ./bin/elasticsearch ./src/elasticsearch

clean:
	rm -f bin/*

test:
	go test -mod=vendor $$(go list github.com/alphagov/paas-prometheus-endpoints/pkg/...)
	go test -mod=vendor ./...
