.PHONY: dev build run migrate create-db

dev:
	go run main.go

build:
	go build -o bin/backend main.go

run:
	./bin/backend

migrate:
	go run cmd/migrate/main.go

create-db:
	go run cmd/create-db/main.go

