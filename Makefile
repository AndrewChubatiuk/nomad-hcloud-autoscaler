SHELL = bash
default: lint check test build

GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),+CHANGES)

GO_LDFLAGS := "-X github.com/AndrewChubatiuk/nomad-hcloud-autoscaler/version.GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)"

# Attempt to use gotestsum for running tests, otherwise fallback to go test.
GO_TEST_CMD = $(if $(shell command -v gotestsum 2>/dev/null),gotestsum --,go test)

.PHONY: tools
tools: lint-tools test-tools

.PHONY: test-tools
test-tools: ## Install the tools used to run tests
	@echo "==> Installing test tools..."
	GO111MODULE=on go install gotest.tools/gotestsum@v1.7.0
	@echo "==> Done"

.PHONY: lint-tools
lint-tools: ## Install the tools used to lint
	@echo "==> Installing lint tools..."
	GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
	GO111MODULE=on go install honnef.co/go/tools/cmd/staticcheck@2021.1.2
	GO111MODULE=on go install github.com/hashicorp/go-hclog/hclogvet@v1.0.0
	@echo "==> Done"

.PHONY: build
build:
	@echo "==> Building HCloud autoscaler..."
	@CGO_ENABLED=0 GO111MODULE=on \
	go build \
	-ldflags $(GO_LDFLAGS) \
	-o ./bin/nomad-hcloud-autoscaler
	@echo "==> Done"

.PHONY: lint
lint: ## Lint the source code
	@echo "==> Linting source code..."
	@golangci-lint run -j 1
	@staticcheck ./...
	@hclogvet .
	@echo "==> Done"

.PHONY: check
check: check-root-mod

.PHONY: check-root-mod
check-root-mod: ## Checks the root Go mod is tidy
	@echo "==> Checking Go mod and Go sum..."
	@GO111MODULE=on go mod tidy
	@if (git status --porcelain | grep -Eq "go\.(mod|sum)"); then \
		echo go.mod or go.sum needs updating; \
		git --no-pager diff go.mod; \
		git --no-pager diff go.sum; \
		exit 1; fi
	@echo "==> Done"

.PHONY: test
test: ## Test the source code
	@echo "==> Testing source code..."
	@$(GO_TEST_CMD) -v -race -cover ./...
	@echo "==> Done"

.PHONY: clean
clean:
	@echo "==> Cleaning build artifacts..."
	@rm -f ./bin/nomad-hcloud-autoscaler
	@echo "==> Done"
