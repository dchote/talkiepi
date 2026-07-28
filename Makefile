# talkiepi Makefile
REPO_SLUG ?= dchote/talkiepi
GORELEASER_IMAGE ?= ghcr.io/goreleaser/goreleaser-cross:v1.25.9

.PHONY: frontend build-deb release-snapshot fmt fmt-check vet test check build run

GO_FILES = $(shell find . -name '*.go' -not -path './research/*' -not -path './frontend/node_modules/*')
GO_PKGS = ./cmd/... ./internal/... ./pkg/...

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@unformatted=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed for:"; echo "$$unformatted"; exit 1; fi

vet:
	go vet $(GO_PKGS)

test:
	CGO_ENABLED=1 go test -race -timeout=60s $(GO_PKGS)

check: fmt-check vet test

frontend:
	cd frontend && yarn install --frozen-lockfile && yarn build
	rm -rf cmd/talkiepi/frontend-dist
	cp -r frontend/dist cmd/talkiepi/frontend-dist

build:
	SKIP_FRONTEND=true ./scripts/build.sh

run:
	CGO_ENABLED=1 go run ./cmd/talkiepi -config configs/talkiepi.yaml -frontend-embed=false

build-deb: frontend
	SKIP_FRONTEND=1 ./scripts/build-deb.sh

release-snapshot: build-deb
