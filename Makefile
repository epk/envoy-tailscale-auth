export GOPRIVATE=buf.build/gen/go/*
export KO_DOCKER_REPO=ghcr.io/epk

BINDIR=$(shell pwd)/bin
DIST=$(shell pwd)/dist

# Tools
GCI=$(BINDIR)/gci
KO=$(BINDIR)/ko
GOFUMPT=$(BINDIR)/gofumpt
GOLANGCI_LINT=$(BINDIR)/golangci-lint
GOTESTSUM=$(BINDIR)/gotestsum

.PHONY: clean
clean:
	rm -rf $(BINDIR)
	rm -rf $(DIST)

.PHONY: fmt
fmt: $(GOFUMPT) $(GCI)
	$(GOFUMPT) -w .
	$(GCI) write . -s Standard -s "Prefix(buf.build)" -s Default -s "Prefix(github.com/epk)"

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --verbose

.PHONY: test
test: $(GOTESTSUM)
	$(GOTESTSUM) -- -race ./...

.PHONY: image
image: $(KO)
	$(KO) build --platform=linux/amd64,linux/arm64 --base-import-paths --image-label \
	org.opencontainers.image.source=https://github.com/epk/envoy-tailscale-auth  \
	--tags=$(shell git rev-parse HEAD) ./cmd/envoy-tailscale-auth

.PHONY: binaries
binaries: $(DIST)
	@for GOOS in darwin linux;																																																			\
	do 																																																															\
		for GOARCH in amd64 arm64;																																																		\
		do 																																																														\
			GOOS=$$GOOS GOARCH=$$GOARCH go build -trimpath -o $(DIST)/envoy-tailscale-auth-$$GOOS-$$GOARCH ./cmd/envoy-tailscale-auth;	\
		done																																																													\
	done

.PHONY: $(DIST)
$(DIST):
	mkdir -p $(DIST)

.PHONY: $(BINDIR)
$(BINDIR):
	mkdir -p $(BINDIR)

.PHONY: $(GCI)
$(GCI): $(BINDIR)
	GOBIN=$(BINDIR) go install github.com/daixiang0/gci

.PHONY: $(KO)
$(KO): $(BINDIR)
	GOBIN=$(BINDIR) go install github.com/google/ko

.PHONY: $(GOFUMPT)
$(GOFUMPT): $(BINDIR)
	GOBIN=$(BINDIR) go install mvdan.cc/gofumpt

.PHONY: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(BINDIR)
	GOBIN=$(BINDIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: $(GOTESTSUM)
$(GOTESTSUM): $(BINDIR)
	GOBIN=$(BINDIR) go install gotest.tools/gotestsum
