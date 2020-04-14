PKG    =  github.com/sapcc/image-migration-dashboard
PREFIX := /usr

# NOTE: This repo uses Go modules, and uses a synthetic GOPATH at
# $(CURDIR)/.gopath that is only used for the build cache. $GOPATH/src/ is
# empty.
GO            := GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR)/build go
GO_BUILDFLAGS :=
GO_LDFLAGS    := -s -w

all: build/image-migration-dashboard

build/image-migration-dashboard: FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'

install: FORCE all
	install -D -m 0755 build/image-migration-dashboard "$(DESTDIR)$(PREFIX)/bin/image-migration-dashboard"

check: FORCE all
	golangci-lint run --enable-all --disable=gochecknoglobals,wsl

clean: FORCE
	rm -rf build/*

vendor: FORCE
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify

update-deps: FORCE
	$(GO) get -u ./...
	make vendor

.PHONY: FORCE
