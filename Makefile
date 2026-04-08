APP_NAME := go-template
BIN      := bin/$(APP_NAME)

.PHONY: run build test coverage docker

run: build
	@$(BIN) http

build:
	@go mod tidy
	@go build -o $(BIN) main.go

test:
	@go fmt ./...
	@go vet ./...
	@go test -v -coverprofile=coverage.out ./...

coverage:
	@go tool cover -html=coverage.out

docker:
	@docker compose up --build

engine:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/docker/$(APP_NAME) main.go
