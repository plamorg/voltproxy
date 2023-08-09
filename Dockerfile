FROM golang:1.21-alpine3.17 AS builder

WORKDIR /usr/src/voltproxy

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/voltproxy

CMD ["voltproxy"]
