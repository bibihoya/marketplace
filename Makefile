.DEFAULT_GOAL := help

.PHONY: help generate generate-commerce generate-user build vet fmt-check compose-config verify \
	run-commerce run-user up down ps logs db-commerce db-user reset

help:
	@echo "Доступные команды:"
	@echo "  make generate       сгенерировать HTTP API обоих сервисов"
	@echo "  make build          собрать оба Go-сервиса"
	@echo "  make vet            запустить go vet для обоих сервисов"
	@echo "  make fmt-check      проверить форматирование Go-кода"
	@echo "  make verify         выполнить все статические проверки"
	@echo "  make run-commerce   запустить Commerce Service без Docker"
	@echo "  make run-user       запустить User Service без Docker"
	@echo "  make up             собрать и поднять проект в Docker"
	@echo "  make down           остановить проект"
	@echo "  make ps             показать состояние контейнеров"
	@echo "  make logs           показать логи HTTP-сервисов"
	@echo "  make db-commerce    открыть psql для Commerce DB"
	@echo "  make db-user        открыть psql для User DB"
	@echo "  make reset          остановить проект и удалить данные БД"

generate: generate-commerce generate-user

generate-commerce:
	cd services/commerce-service && go generate ./...

generate-user:
	cd services/user-service && go generate ./...

build: generate
	cd services/commerce-service && go build ./...
	cd services/user-service && go build ./...

vet: generate
	cd services/commerce-service && go vet ./...
	cd services/user-service && go vet ./...

fmt-check: generate
	@files="$$(find services -type f -name '*.go' -print)"; \
	unformatted="$$(gofmt -l $$files)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Нужно отформатировать:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

compose-config:
	docker compose config --quiet

verify: generate fmt-check build vet compose-config

run-commerce: generate-commerce
	cd services/commerce-service && go run ./cmd/server

run-user: generate-user
	cd services/user-service && go run ./cmd/server

up:
	docker compose up --build -d

down:
	docker compose down

ps:
	docker compose ps -a

logs:
	docker compose logs --tail=100 -f commerce-service user-service

db-commerce:
	docker compose exec commerce-db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

db-user:
	docker compose exec user-db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

reset:
	docker compose down -v
