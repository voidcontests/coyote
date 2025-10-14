FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

RUN apk add --no-cache git # purely for baking commit and branch into executable

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -a -ldflags="-w -s \
    -X runner/internal/version.GIT_COMMIT=$(git rev-parse --short HEAD) \
    -X runner/internal/version.GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)" \
    -o build/coyote ./cmd/coyote

# Lightweight docker container with binaries only
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache docker-cli

COPY --from=builder /app/build ./build
COPY --from=builder /app/seccomp.json ./seccomp.json

CMD ["./build/coyote"]
