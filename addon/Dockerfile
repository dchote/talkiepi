# Multi-stage build for Home Assistant addon. Build with context at repo root:
#   docker build -f addon/Dockerfile --build-arg BUILD_FROM=ghcr.io/... .
ARG BUILD_FROM=ghcr.io/hassio-addons/debian-base/amd64:9.2.0
FROM node:20-bookworm-slim AS frontend
WORKDIR /app/frontend

COPY frontend/package.json frontend/yarn.lock* ./
RUN yarn install --frozen-lockfile || yarn install

COPY frontend/ ./
RUN yarn build

FROM golang:1.25-bookworm AS builder
WORKDIR /app

ENV CGO_ENABLED=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config \
    libopus-dev \
    libspeexdsp-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/frontend/dist ./cmd/talkiepi/frontend-dist

RUN go build -ldflags "-s -w" -tags opus,speexdsp,nolibopusfile,embed_frontend -o talkiepi ./cmd/talkiepi

FROM ${BUILD_FROM}

RUN apt-get update && apt-get install -y --no-install-recommends \
    libopus0 \
    libspeexdsp1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/talkiepi /usr/bin/talkiepi
COPY addon/rootfs /
