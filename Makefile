.PHONY: build test login reset-waiting migrate docker

build:
	go build -o bin/bot.exe ./cmd/bot
	go build -o bin/login.exe ./cmd/login
	go build -o bin/reset-waiting.exe ./cmd/resetwaiting
	go build -o bin/migrate.exe ./cmd/migrate

test:
	go test ./...

login:
	go run ./cmd/login

reset-waiting:
	go run ./cmd/resetwaiting

migrate:
	go run ./cmd/migrate

docker:
	docker build -t locatorbpla .
