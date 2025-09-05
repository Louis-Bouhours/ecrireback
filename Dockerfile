# syntax=docker/dockerfile:1

FROM golang:1.24.3-bookworm AS builder
WORKDIR /src
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Chemin du package main
ARG APP_PATH=./cmd.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

# Build du binaire (avec vérification du chemin)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    echo "Building main package at: ${APP_PATH}" && \
    if [ ! -d "${APP_PATH}" ]; then \
      echo "ERROR: '${APP_PATH}' n'existe pas sous /src."; \
      echo "Arborescence (maxdepth 2) sous /src:"; \
      find /src -maxdepth 2 -type d | sed 's|^/src||'; \
      exit 1; \
    fi && \
    go build -ldflags="-s -w" -o /out/app "${APP_PATH}"

FROM debian:bookworm-slim AS runtime
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/app /app/app

# L'app lit APP_PORT
ENV GIN_MODE=release \
    APP_PORT=8081

EXPOSE 8081

USER 65532:65532
CMD ["/app/app"]