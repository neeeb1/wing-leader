include .env
export

BINARY_NAME=go-app

build-app:
	tailwindcss -i web/style.css -o web/output.css
	go build -o ./bin/${BINARY_NAME} main.go

run:
	make build
	./bin/${BINARY_NAME}

clean:
	go clean
	rm ./bin/${BINARY_NAME}

compose-up:
	tailwindcss -i web/style.css -o web/output.css
	docker compose -f ./build/docker-compose.yml up --build -d

compose-down:
	docker compose -f ./build/docker-compose.yml down

sqlc:
	sqlc generate

goose-up:
	goose -dir ./sql/schema/ postgres "$(DB_URL)" up

goose-down:
	goose -dir ./sql/schema/ postgres "$(DB_URL") down
