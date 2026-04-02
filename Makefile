APP := pmp
PKG := ./cmd/pmp
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: fmt test build clean tidy snapshot

fmt:
	go fmt ./...

test:
	go test ./...

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP) $(PKG)

tidy:
	go mod tidy

clean:
	rm -f $(APP) $(APP).exe coverage.out

snapshot:
	goreleaser build --snapshot --clean
