# syntax=docker/dockerfile:1

# Étape 1: Build binaire Go
FROM golang:1.22-bookworm AS builder
WORKDIR /src

# Modules en cache pour accélérer
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Source
COPY . .

# Ajuste le chemin du package main si nécessaire (ex: ./cmd/api)
# Ici on suppose un main.go à la racine du repo
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/app .

# Étape 2: Runtime minimal
FROM debian:bookworm-slim AS runtime
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata wget \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/app /app/app

# Variables par défaut (peuvent être surchargées par .env)
ENV GIN_MODE=release \
    PORT=8081

EXPOSE 8081


# Utilisateur non-root
USER 65532:65532

CMD ["/app/app"]