# Common development tasks. Run `make` (or `make build`) to build ./aiterm.

GO            ?= go
GOLANGCI_LINT ?= golangci-lint
GORELEASER    ?= goreleaser

BINARY  := aiterm
VERSION ?= $(patsubst v%,%,$(shell git describe --tags --always --dirty 2>/dev/null || echo dev))

.DEFAULT_GOAL := build

.PHONY: build test cover fuzz lint fmt tidy vuln snapshot clean

# build: build ./aiterm with the version from git describe
build:
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

# test: run the tests with the race detector
test:
	$(GO) test -race ./...

# cover: run the tests and print coverage per function (writes coverage.out)
cover:
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

# fuzz: fuzz the model reply validation for 30 seconds
fuzz:
	$(GO) test -run='^$$' -fuzz=FuzzValidateCmd -fuzztime=30s .

# lint: run golangci-lint (https://golangci-lint.run)
lint:
	$(GOLANGCI_LINT) run

# fmt: format the code with gofmt and goimports through golangci-lint
fmt:
	$(GOLANGCI_LINT) fmt

# tidy: tidy go.mod and go.sum
tidy:
	$(GO) mod tidy

# vuln: check for known vulnerabilities with govulncheck
vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...

# snapshot: build release archives locally into dist/ without publishing
snapshot:
	$(GORELEASER) release --snapshot --clean

# clean: remove build output
clean:
	rm -rf $(BINARY) dist coverage.out
