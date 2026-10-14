.PHONY: generate build run

generate:
	cd services/commerce-service && go generate ./...

build: generate
	cd services/commerce-service && go build ./...

run: generate
	cd services/commerce-service && go run ./cmd/server
