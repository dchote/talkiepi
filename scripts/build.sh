#!/bin/bash
set -e
cd "$(dirname "$0")/.."

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}"
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X main.version=${VERSION}"
LDFLAGS="${LDFLAGS} -X main.commit=${COMMIT}"
LDFLAGS="${LDFLAGS} -X main.buildTime=${BUILD_TIME}"

SKIP_FRONTEND="${SKIP_FRONTEND:-false}"
# Always enable real Opus + SpeexDSP when CGO is available.
# nolibopusfile: encode/decode only (no libopusfile).
TAGS="opus,speexdsp,nolibopusfile"
if [ "${SKIP_FRONTEND}" != "true" ] && [ "${SKIP_FRONTEND}" != "1" ] && [ -f frontend/package.json ]; then
    echo "Building frontend..."
    cd frontend && yarn build && cd ..

    echo "Copying frontend dist to embed location..."
    rm -rf cmd/talkiepi/frontend-dist
    cp -r frontend/dist cmd/talkiepi/frontend-dist
    TAGS="${TAGS},embed_frontend"
elif [ -d "cmd/talkiepi/frontend-dist" ] && [ -n "$(ls -A cmd/talkiepi/frontend-dist 2>/dev/null)" ]; then
    TAGS="${TAGS},embed_frontend"
fi

mkdir -p build

echo "Building talkiepi ${VERSION} (${COMMIT}) tags=${TAGS}..."

CGO_ENABLED=1 go build -tags "${TAGS}" -ldflags "${LDFLAGS}" -o build/talkiepi ./cmd/talkiepi

echo "Done. Binary at build/talkiepi"
