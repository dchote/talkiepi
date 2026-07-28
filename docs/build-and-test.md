# Build and test

## Prerequisites

### macOS (Homebrew)

```bash
brew install go opus speexdsp pkg-config
# Frontend
brew install node yarn   # or enable corepack
```

Ensure `pkg-config` can find Opus and SpeexDSP (`brew --prefix opus` / `speexdsp`).

### Debian / Ubuntu

```bash
sudo apt-get update
sudo apt-get install -y golang-go pkg-config libopus-dev libspeexdsp-dev
# Node 20+ and yarn for frontend builds
```

### All platforms

- **Go 1.25+** (module declares `go 1.25.5`)
- **CGO_ENABLED=1** for Opus and SpeexDSP
- **Docker** for cross `.deb` builds (required on macOS)
- **QEMU** for arm64 containers on amd64 hosts (`docker run --platform linux/arm64`)

## Quick commands

| Goal | Command |
|------|---------|
| Format check + vet + tests | `make check` |
| Format only | `make fmt` / `make fmt-check` |
| Vet | `make vet` |
| Test (race, 60s) | `make test` |
| Frontend + embed copy | `make frontend` |
| Binary without frontend | `SKIP_FRONTEND=true ./scripts/build.sh` or `make build` |
| Dev API (no embed) | `make run` |
| Debian packages | `make build-deb` or `./scripts/build-deb.sh` |

## `make check`

```bash
make check   # fmt-check && vet && test
```

Tests run:

```bash
CGO_ENABLED=1 go test -race -timeout=60s ./cmd/... ./internal/... ./pkg/...
```

Do **not** start the talkiepi process to validate—listeners and audio devices will block. Prefer `go build` / `go test`.

## CGO

Opus and SpeexDSP require CGO. Always set `CGO_ENABLED=1` for build and test unless you intentionally use stub builds. Missing headers produce link errors; install the brew/apt packages above.

## Frontend and `SKIP_FRONTEND`

Full release binaries use `-tags embed_frontend` and need `cmd/talkiepi/frontend-dist/` populated:

```bash
make frontend
# or
./scripts/build.sh          # builds frontend unless SKIP_FRONTEND is set
```

Skip frontend when iterating on Go only:

```bash
SKIP_FRONTEND=true ./scripts/build.sh
# or
make build
```

**Never** pass `-tags embed_frontend` if `frontend-dist` is missing or empty—`//go:embed` will fail.

CI runs `yarn build` to verify the frontend compiles; unit tests do **not** require an embedded frontend.

## Debian packages via Docker (including macOS)

[`scripts/build-deb.sh`](../scripts/build-deb.sh) builds amd64 and arm64 `.deb` files with [goreleaser-cross](https://github.com/goreleaser/goreleaser-cross) `v1.25.9`, installing `libopus-dev` and `libspeexdsp-dev` inside each container so CGO matches the target arch.

```bash
# Full: frontend + both arches
./scripts/build-deb.sh

# Reuse existing frontend-dist
SKIP_FRONTEND=1 ./scripts/build-deb.sh
```

Outputs: `dist/*.deb`.

On Apple Silicon, Docker Desktop must allow multi-platform / QEMU for `linux/amd64` and `linux/arm64`.

Goreleaser configs: [`.goreleaser-amd64.yaml`](../.goreleaser-amd64.yaml), [`.goreleaser-arm64.yaml`](../.goreleaser-arm64.yaml).

## Dev loop

```bash
# Terminal 1 — API
make run

# Terminal 2 — UI with HMR
cd frontend && yarn install && yarn dev
```

Default config: [`configs/talkiepi.yaml`](../configs/talkiepi.yaml).
