.PHONY: clean test

APP_ROOT ?= $(PWD)

bin/redis: clean
	go build -o ./bin/redis ./src/redis

clean:
	rm -f bin/redis

test:
	go test -mod=vendor $$(go list github.com/alphagov/paas-prometheus-endpoints/pkg/...)
	go test ./...
