GOLANGCI_LINT_CACHE?=/tmp/cache
COMPOSE_FILE=docker-compose.yml

.PHONY: build
build:
	docker-compose -f $(COMPOSE_FILE) build

.PHONY: up
up:
	docker-compose -f $(COMPOSE_FILE) up

.PHONY: down
down:
	docker-compose -f $(COMPOSE_FILE) down

.PHONY: swag
swag:
	swag init -g internal/server/server.go

.PHONY: lint
lint: _golangci-lint-rm-unformatted-report

.PHONY: _golangci-lint-reports-mkdir
_golangci-lint-reports-mkdir:
	mkdir -p ./golangci-lint

.PHONY: _golangci-lint-run
_golangci-lint-run: _golangci-lint-reports-mkdir
	-docker run --rm \
	-v $(shell pwd):/app \
	-v $(GOLANGCI_LINT_CACHE):/root/.cache \
	-w /app \
	golangci/golangci-lint:v1.64.5 \
		golangci-lint run \
			-c .golangci.yml \
	> ./golangci-lint/report-unformatted.json

.PHONY: _golangci-lint-format-report
_golangci-lint-format-report: _golangci-lint-run
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json

.PHONY: _golangci-lint-rm-unformatted-report
_golangci-lint-rm-unformatted-report: _golangci-lint-format-report
	rm ./golangci-lint/report-unformatted.json

.PHONY: golangci-lint-clean
golangci-lint-clean:
	sudo rm -rf ./golangci-lint