# syntax=docker/dockerfile:1

# Étape 1: Build binaire Go (aligne avec go.mod >= 1.24.3)
FROM golang:1.24.3-bookworm AS builder
WORKDIR /src

# On a déjà la bonne toolchain
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Chemin du package main (override si nécessaire)
ARG APP_PATH=./cmd/api
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

# Build du binaire
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    echo "Building main package at: ${APP_PATH}" && \
    go build -ldflags="-s -w" -o /out/app "${APP_PATH}"

# Étape 2: Runtime minimal
FROM debian:bookworm-slim AS runtime
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata wget \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/app /app/app

ENV GIN_MODE=release \
    PORT=8081

EXPOSE 8081


# Non-root
USER 65532:65532

CMD ["/app/app"]