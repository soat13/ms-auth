.PHONY: install up down run test test-coverage mod vendor tidy fmt lint sh \
        migrate-up migrate-down migrate-status seed-up sonar sonar-analysis \
        aws-create-table aws-seed-user aws-integration-test

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Infra
up:
	@if [ ! -f .env ]; then cp .env-example .env; fi
	docker compose up -d

down:
	docker compose down

install:
	@if [ ! -f .env ]; then cp .env-example .env; fi
	docker compose down -v --remove-orphans
	docker compose pull
	docker compose up -d --build --force-recreate

run:
	docker compose exec app-dev go run ./cmd/api

# QA
test:
	docker compose exec app-dev go test ./...
test-coverage:
	docker compose exec app-dev go test -coverpkg=./internal/...,./pkg/... -coverprofile=coverage.out ./...
mod:
	docker compose exec app-dev go mod tidy
vendor:
	docker compose exec app-dev go mod vendor
tidy:
	docker compose exec app-dev go mod tidy
fmt:
	docker compose exec app-dev go fmt ./...
lint:
	docker compose exec app-dev go vet ./...

# Shell
sh:
	docker compose exec app-dev bash

# AWS / Local Testing
DYNAMODB_ENDPOINT ?= http://localhost:8000

aws-create-table:
	@chmod +x scripts/aws/create-dynamodb-table.sh
	./scripts/aws/create-dynamodb-table.sh --endpoint $(DYNAMODB_ENDPOINT)

aws-seed-user:
	@chmod +x scripts/aws/seed-test-user.sh
	./scripts/aws/seed-test-user.sh --endpoint $(DYNAMODB_ENDPOINT)

aws-integration-test:
	@chmod +x scripts/aws/integration-test.sh
	./scripts/aws/integration-test.sh --endpoint $(DYNAMODB_ENDPOINT)

# -------------------------------
# SonarQube Analysis
# -------------------------------

sonar: up
	@if [ -z "$(SONAR_TOKEN)" ]; then \
		echo "Error: SONAR_TOKEN is not set. Please set it in your .env file."; \
		exit 1; \
	fi
	make test-coverage
	docker compose run --rm sonar-scanner \
		-Dsonar.projectKey=ms-auth \
		-Dsonar.projectName="MS-Auth" \
		-Dsonar.sources=. \
		-Dsonar.exclusions=**/vendor/**,**/mocks/**,**/*_test.go,**/tests/**,**/scripts/**,**/assets/**,.env*,**/*.md,**/cmd/**,**/views.go,**/repository.go,**/bootstrap/** \
		-Dsonar.tests=. \
		-Dsonar.test.inclusions=**/*_test.go \
		-Dsonar.go.coverage.reportPaths=coverage.out