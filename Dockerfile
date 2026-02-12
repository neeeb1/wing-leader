# Use Go 1.25 as base image
FROM golang:1.25-bookworm AS base

WORKDIR /

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o go-app

EXPOSE 8080

CMD ["/build/go-app"]

