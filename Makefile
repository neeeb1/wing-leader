include .env
export

BINARY_NAME=go-app

clean:
	go clean
	rm ./bin/${BINARY_NAME}

compose-up:
	tailwindcss -i web/static/style.css -o web/static/output.css
	docker compose -f ./build/docker-compose.yml up --build -d

compose-down:
	docker compose -f ./build/docker-compose.yml down

compose-restart:
	make compose-down
	make compose-up
