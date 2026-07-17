GO ?= go
PLUGIN_ID := codex-model-discovery

.PHONY: build fmt test test-race vet clean

build:
	mkdir -p build
	CGO_ENABLED=1 $(GO) build -trimpath -buildmode=c-shared -o build/$(PLUGIN_ID).so .

fmt:
	$(GO)fmt -w .

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf build dist
