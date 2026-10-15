.PHONY: generate generate-commerce generate-user build run-commerce run-user

generate: generate-commerce generate-user

generate-commerce:
	cd services/commerce-service && go generate ./...

generate-user:
	cd services/user-service && go generate ./...

build: generate
	cd services/commerce-service && go build ./...
	cd services/user-service && go build ./...

run-commerce: generate-commerce
	cd services/commerce-service && go run ./cmd/server

run-user: generate-user
	cd services/user-service && go run ./cmd/server
