LDFLAGS ?=

# Print the help menu.
.PHONY: help
help:
	@echo "ingress2eg - Convert Ingress resources to Gateway API and Envoy Gateway CRD"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: vet fmt test build  ## Run all checks and build

.PHONY: fmt
fmt:  ## Run go fmt against code
	gofmt -w  ./cmd

.PHONY: vet
vet:  ## Run go vet against code
	go vet ./cmd/...

.PHONY: test
test: vet  ## Run go test against code
	go test -race -cover ./cmd/...

.PHONY: yaml-test
yaml-test: vet  ## Run yaml tests
	go test -v ./test/yaml/...

.PHONY: yaml-test-override
yaml-test-override: vet  ## Run yaml tests - override golden files
	go test -v ./test/yaml/... -override

.PHONY: build
build: vet  ## Build the binary
	go build $(LDFLAGS) -o ingress2eg .

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION = v2.7.2

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
