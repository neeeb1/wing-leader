include .env
export

BINARY_NAME=go-app

clean:
	go clean
	rm ./bin/${BINARY_NAME}

compose-up:
	tailwindcss -i web/style.css -o web/output.css
	docker compose -f ./build/docker-compose.yml up --build -d

compose-down:
	docker compose -f ./build/docker-compose.yml down
