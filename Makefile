BINARY := bin/goname
GO ?= go
GOCACHE ?= $(CURDIR)/.cache/go-build
export GOCACHE

.PHONY: all build test verify fmt vet clean install

all: build

build:
	@mkdir -p $(dir $(BINARY))
	$(GO) build -trimpath -o $(BINARY) ./cmd/goname

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

verify: fmt vet test build
	@$(BINARY) --help >/dev/null
	@test "$$($(BINARY) --words 3 | awk -F- '{print NF}')" = 3

install:
	$(GO) install ./cmd/goname

clean:
	$(GO) clean -testcache
	rm -rf bin dist .cache
