# talkiepi Makefile
# Requires: Docker, Node 20+ and yarn (for frontend), Homebrew opus/speexdsp/pkg-config on macOS.

REPO_SLUG ?= dchote/talkiepi
GORELEASER_IMAGE ?= ghcr.io/goreleaser/goreleaser-cross:v1.25.9

.PHONY: frontend build-deb release-snapshot fmt fmt-check vet test check build run

GO_FILES = $(shell find . -name '*.go' -not -path './research/*' -not -path './frontend/node_modules/*')
GO_PKGS = ./cmd/... ./internal/... ./pkg/...
# Real Opus + SpeexDSP require CGO and these tags (stubs are used without them).
# nolibopusfile avoids a libopusfile dependency (we only encode/decode PCM).
GO_TAGS ?= opus,speexdsp,nolibopusfile

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@unformatted=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed for:"; echo "$$unformatted"; exit 1; fi

vet:
	CGO_ENABLED=1 go vet -tags $(GO_TAGS) $(GO_PKGS)

test:
	CGO_ENABLED=1 go test -tags $(GO_TAGS) -race -timeout=60s $(GO_PKGS)

check: fmt-check vet test

frontend:
	cd frontend && yarn install --frozen-lockfile && yarn build
	rm -rf cmd/talkiepi/frontend-dist
	cp -r frontend/dist cmd/talkiepi/frontend-dist

build:
	SKIP_FRONTEND=true ./scripts/build.sh

# Serve API + prebuilt UI from frontend/dist on :8080.
# Builds a binary first so Ctrl+C is handled by talkiepi (not go run/make noise).
# For hot-reload UI: make run in one terminal, and `cd frontend && yarn dev` in another.
run:
	@test -f frontend/dist/index.html || (echo "frontend/dist missing; building..." && $(MAKE) frontend)
	CGO_ENABLED=1 go build -tags $(GO_TAGS) -o bin/talkiepi ./cmd/talkiepi
	@# Ignore SIGINT in the recipe shell so make is not aborted with Error 1 while
	@# talkiepi handles Ctrl+C and exits 0 after graceful shutdown.
	@bash -c 'trap "" INT; ./bin/talkiepi -config configs/talkiepi.yaml -frontend-embed=false -frontend-dir=frontend/dist; exit $$?'

# Build .deb packages locally via Docker (same as CI). Works on macOS and Linux.
# Produces dist/*.deb (amd64 and arm64). Requires Docker.
build-deb: frontend
	SKIP_FRONTEND=1 ./scripts/build-deb.sh

# Alias for build-deb (same behavior).
release-snapshot: build-deb
