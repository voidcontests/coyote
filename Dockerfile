FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./build/coyote ./cmd/coyote

# Lightweight docker container with binaries only
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache docker-cli

COPY --from=builder /app/build ./build
COPY --from=builder /app/seccomp.json ./seccomp.json

CMD ["./build/coyote"]
