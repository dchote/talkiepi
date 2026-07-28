#!/usr/bin/env bash
# Build .deb packages locally via Docker (same as CI). Works on Linux and macOS.
# Produces dist/*.deb (amd64 and arm64). Requires Docker, Node 20+, and yarn.
#
# Builds each architecture in a matching container (amd64 in linux/amd64, arm64 in
# linux/arm64) so CGO uses the correct toolchain, then merges dist outputs.
#
# Usage:
#   ./scripts/build-deb.sh              # build frontend + .deb
#   SKIP_FRONTEND=1 ./scripts/build-deb.sh   # skip frontend (use existing frontend-dist)
set -e

REPO_SLUG="${REPO_SLUG:-dchote/talkiepi}"
GORELEASER_IMAGE="${GORELEASER_IMAGE:-ghcr.io/goreleaser/goreleaser-cross:v1.25.9}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

if [ "${SKIP_FRONTEND}" != "1" ] && [ "${SKIP_FRONTEND}" != "true" ]; then
  echo "Building frontend..."
  (cd frontend && yarn install --frozen-lockfile && yarn build)
  echo "Copying frontend to embed location..."
  rm -rf cmd/talkiepi/frontend-dist
  cp -r frontend/dist cmd/talkiepi/frontend-dist
else
  if [ ! -d "cmd/talkiepi/frontend-dist" ] || [ -z "$(ls -A cmd/talkiepi/frontend-dist 2>/dev/null)" ]; then
    echo "error: SKIP_FRONTEND is set but cmd/talkiepi/frontend-dist is missing or empty. Run without SKIP_FRONTEND first." >&2
    exit 1
  fi
  echo "Skipping frontend build (using existing frontend-dist)."
fi

run_goreleaser() {
  local config="$1"
  local platform="$2"
  # Docker prep installs CGO deps, then runs goreleaser (image ENTRYPOINT is goreleaser).
  docker run --rm --privileged --platform "$platform" \
    --entrypoint bash \
    -v "$ROOT":/go/src/github.com/$REPO_SLUG \
    -w /go/src/github.com/$REPO_SLUG \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e CGO_ENABLED=1 \
    -e GORELEASER_CURRENT_TAG=v0.0.0-snapshot \
    "$GORELEASER_IMAGE" \
    -c "apt-get update && apt-get install -y pkg-config libopus-dev libspeexdsp-dev && goreleaser release --snapshot --skip=publish --clean --parallelism 1 -f ${config}"
}

echo "Building .deb for linux/amd64..."
run_goreleaser .goreleaser-amd64.yaml linux/amd64
mv dist dist-amd64

echo "Building .deb for linux/arm64..."
run_goreleaser .goreleaser-arm64.yaml linux/arm64
mv dist dist-arm64

echo "Merging dist outputs..."
mkdir -p dist
cp -r dist-amd64/* dist/ 2>/dev/null || true
cp -r dist-arm64/* dist/ 2>/dev/null || true
rm -rf dist-amd64 dist-arm64

echo "Cleaning up: keeping only .deb packages in dist/"
find dist -mindepth 1 -maxdepth 1 ! -name '*.deb' -exec rm -rf {} +

echo "Done. Debian packages: dist/*.deb"
