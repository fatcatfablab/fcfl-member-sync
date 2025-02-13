# Original from https://github.com/crazywolf132/ultimate-gomake/
# Heavily modified
#
# =============================================================================
# 🎯 Project Configuration
# =============================================================================
# Project Settings
PROJECT_NAME ?= fcfl-member-sync
ORGANIZATION ?= fatcatfablab
DESCRIPTION ?= "FatCatFabLab's member syncing facilities"
MAINTAINER ?= "self@miquelruiz.net"

# Feature Flags (can be disabled in .env)
ENABLE_DOCKER ?= false
ENABLE_PROTO ?= true
ENABLE_DOCS ?= false
ENABLE_SECURITY_SCAN ?= true
ENABLE_ADVANCED_TESTING ?= true
ENABLE_BUILD_CACHE ?= true

# Project Structure
PROJECT_TYPE ?= basic # basic, monorepo, microservices
MONOREPO_SERVICES ?= $(wildcard services/*)
BUILD_TARGETS ?= cmd/client cmd/server cmd/report

# Version Control
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILT_BY ?= $(shell whoami)

# Go Configuration
GO ?= go
GOCMD = $(shell which go)
GOPATH ?= $(shell $(GO) env GOPATH)
GOBIN ?= $(GOPATH)/bin
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
CGO_ENABLED ?= 0

# Tools & Linters
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GOFUMPT ?= $(GOBIN)/gofumpt
GODOC ?= $(GOBIN)/godoc
GOVULNCHECK ?= $(GOBIN)/govulncheck
MOCKGEN ?= $(GOBIN)/mockgen
AIR ?= $(GOBIN)/air
SHELL = /bin/bash

# Directories
ROOT_DIR ?= .
BIN_DIR ?= $(ROOT_DIR)/bin
DIST_DIR ?= $(ROOT_DIR)/dist
DOCS_DIR ?= $(ROOT_DIR)/docs
TOOLS_DIR ?= $(ROOT_DIR)/tools
PROTO_DIR ?= $(ROOT_DIR)/proto
CONFIG_DIR ?= $(ROOT_DIR)/configs
SCRIPTS_DIR ?= $(ROOT_DIR)/scripts
MIGRATIONS_DIR ?= $(ROOT_DIR)/migrations

# Source Files
GOFILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.git/*")
GOPACKAGES = $(shell $(GO) list ./... | grep -v /vendor/)

# Build Configuration
BUILD_TAGS ?= civicrm unifiaccess
EXTRA_TAGS ?=
ALL_TAGS = $(BUILD_TAGS) $(EXTRA_TAGS)

# Linker Flags
LD_FLAGS += -s -w
LD_FLAGS += -X 'main.Version=$(VERSION)'
LD_FLAGS += -X 'main.Commit=$(GIT_COMMIT)'
LD_FLAGS += -X 'main.Branch=$(GIT_BRANCH)'
LD_FLAGS += -X 'main.BuildTime=$(BUILD_TIME)'
LD_FLAGS += -X 'main.BuiltBy=$(BUILT_BY)'

# Performance & Debug Flags
GCFLAGS ?=
ASMFLAGS ?=

# Docker Configuration
DOCKER_REGISTRY ?= docker.io
DOCKER_REPO ?= $(ORGANIZATION)/$(PROJECT_NAME)
DOCKER_IMAGE ?= $(DOCKER_REGISTRY)/$(DOCKER_REPO)
DOCKER_TAG ?= $(VERSION)
DOCKERFILE ?= Dockerfile
DOCKER_BUILD_ARGS ?=
DOCKER_BUILD_CONTEXT ?= .

# Test Configuration
TEST_TIMEOUT ?= 5m
TEST_FLAGS ?= -race -cover
COVERAGE_OUT ?= coverage.out
COVERAGE_HTML ?= coverage.html
COVERAGE_THRESHOLD ?= 80
BENCH_FLAGS ?= -benchmem
BENCH_TIME ?= 2s
TEST_PATTERN ?= .
SKIP_PATTERN ?=

# Cross-Compilation Targets
PLATFORMS ?= \
    linux/amd64 \
    linux/arm/7

# =============================================================================
# 🎨 Terminal Colors & Emoji
# =============================================================================
# Colors
BLUE := \033[34m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
BOLD := \033[1m
RESET := \033[0m

# Status Indicators
INFO := @echo ℹ️
SUCCESS = @echo ✅
WARN := @echo ⚠️
ERROR := @echo ❌
WORKING := @echo 🔨
DEBUG := @echo 🔍
ROCKET := @echo 🚀
PACKAGE := @echo 📦
TRASH := @echo 🗑️

# =============================================================================
# 🎯 Core Build System
# =============================================================================
.PHONY: init
init: ## Initialize project with sensible defaults
	$(INFO) Initializing project...
	@if [ ! -f "go.mod" ]; then \
		$(GO) mod init $(shell basename $(CURDIR)); \
	fi
	@if [ ! -f ".env" ]; then \
		echo "# Project Configuration" > .env; \
		echo "PROJECT_NAME=$(PROJECT_NAME)" >> .env; \
		echo "ENABLE_DOCKER=true" >> .env; \
		echo "ENABLE_PROTO=false" >> .env; \
	fi
	@if [ ! -f ".gitignore" ]; then \
		curl -sL https://www.gitignore.io/api/go > .gitignore; \
	fi
	@mkdir -p \
		main \
		testdata \
		.github/workflows
	@if [ ! -f "main/main.go" ]; then \
		echo 'package main\n\nimport "fmt"\n\nfunc main() {\n    fmt.Println("Hello, World!")\n}' > main/main.go; \
	fi
	$(MAKE) deps
	$(SUCCESS) Project initialized!

.PHONY: build
build: $(BIN_DIR) ## Build all targets
	$(WORKING) Building project...
	@$(foreach target,$(BUILD_TARGETS),$(MAKE) build-target TARGET=$(target);)
	@wait
	$(SUCCESS) Build complete!

.PHONY: build-target
build-target: generate
	$(INFO) Building $(TARGET)...
	@if [ -f "$(TARGET)/Makefile" ]; then \
		$(MAKE) -C $(TARGET) build; \
	else \
		CGO_ENABLED=$(CGO_ENABLED) \
		$(GO) build -tags '$(ALL_TAGS)' \
			-ldflags '$(LD_FLAGS)' \
			-gcflags '$(GCFLAGS)' \
			-asmflags '$(ASMFLAGS)' \
			-o $(BIN_DIR)/$(notdir $(TARGET)) \
			./$(TARGET); \
	fi

.PHONY: install
install: build ## Install the application
	$(WORKING) Installing $(PROJECT_NAME)...
	$(GO) install -tags '$(ALL_TAGS)' -ldflags '$(LD_FLAGS)' ./main
	$(SUCCESS) Installation complete!

# =============================================================================
# 🔄 Development Workflow
# =============================================================================
.PHONY: dev
dev: deps generate ## Start development environment
	$(INFO) Starting development environment...
	@if [ ! -f ".air.toml" ]; then \
		cp $(CONFIG_DIR)/air.toml.example .air.toml 2>/dev/null || \
		curl -sL https://raw.githubusercontent.com/cosmtrek/air/master/air.conf.example > .air.toml; \
	fi
	$(ROCKET) Running with hot reload...
	$(AIR) -c .air.toml

.PHONY: run
run: build ## Run the application
	$(ROCKET) Running $(PROJECT_NAME)...
	$(BIN_DIR)/$(PROJECT_NAME)

.PHONY: generate
generate: ## Run code generation
	$(WORKING) Running code generation...
	$(GO) generate ./...
	@if [ -n "$(wildcard $(PROTO_DIR)/*.proto)" ]; then \
		$(MAKE) proto; \
	fi
	$(SUCCESS) Generation complete!

# =============================================================================
# 🧪 Testing & Quality
# =============================================================================
.PHONY: test
test: ## Run tests
	$(INFO) Running tests...
	$(GO) test $(TEST_FLAGS) \
		-tags '$(ALL_TAGS)' \
		-timeout $(TEST_TIMEOUT) \
		-run '$(TEST_PATTERN)' \
		$(if $(SKIP_PATTERN),-skip '$(SKIP_PATTERN)') \
		./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	$(INFO) Running tests with coverage...
	$(GO) test $(TEST_FLAGS) \
		-timeout $(TEST_TIMEOUT) \
		-coverprofile=$(COVERAGE_OUT) \
		./...
	$(GO) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@coverage=$$(go tool cover -func=$(COVERAGE_OUT) | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$${coverage%.*}" -lt "$(COVERAGE_THRESHOLD)" ]; then \
		$(ERROR) "Coverage $${coverage}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi
	$(SUCCESS) Coverage report generated: $(COVERAGE_HTML)

.PHONY: test-integration
test-integration: ## Run integration tests
	$(INFO) Running integration tests...
	$(GO) test $(TEST_FLAGS) \
		-tags=integration \
		-timeout $(TEST_TIMEOUT) \
		./...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	$(INFO) Running end-to-end tests...
	$(GO) test $(TEST_FLAGS) \
		-tags=e2e \
		-timeout $(TEST_TIMEOUT) \
		./test/e2e/...

.PHONY: bench
bench: ## Run benchmarks
	$(INFO) Running benchmarks...
	$(GO) test -bench=. \
		$(BENCH_FLAGS) \
		-run=^$ \
		-benchtime=$(BENCH_TIME) \
		./...

.PHONY: lint
lint: ## Run linters
	$(INFO) Running linters...
	$(GOLANGCI_LINT) run --fix
	$(SUCCESS) Lint complete!

.PHONY: fmt
fmt: ## Format code
	$(INFO) Formatting code...
	$(GO) fmt ./...
	$(GOFUMPT) -l -w .
	$(SUCCESS) Format complete!

.PHONY: vet
vet: ## Run go vet
	$(INFO) Running go vet...
	$(GO) vet --tags '$(ALL_TAGS)' ./...
	$(SUCCESS) Vet complete!

.PHONY: security
security: ## Run security checks
	$(INFO) Running security checks...
	$(GOVULNCHECK) ./...
	$(SUCCESS) Security check complete!

# =============================================================================
# 🏗️ Build Variations
# =============================================================================
.PHONY: build-all
build-all: $(DIST_DIR) ## Build for all platforms
	$(WORKING) Building for all platforms...
	@rm -Rf $(DIST_DIR)/*
	@$(foreach target,$(BUILD_TARGETS),\
		$(foreach platform,$(PLATFORMS),\
			$(eval OS := $(word 1,$(subst /, ,$(platform)))) \
			$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
			$(eval ARM := $(word 3,$(subst /, ,$(platform)))) \
			$(eval NAME := $(word 2,$(subst /, ,$(target)))) \
			echo "🔨 Building $(target) for $(OS)/$(ARCH)$(if $(ARM),/v$(ARM))..." && \
			GOOS=$(OS) GOARCH=$(ARCH) $(if $(ARM),GOARM=$(ARM)) \
			CGO_ENABLED=$(CGO_ENABLED) \
			$(GO) build -tags '$(ALL_TAGS)' \
				-ldflags '$(LD_FLAGS)' \
				-o $(DIST_DIR)/$(PROJECT_NAME)-$(NAME)-$(OS)-$(ARCH)$(if $(ARM),v$(ARM))$(if $(findstring windows,$(OS)),.exe,) \
				./$(target); \
		) \
	)
	$(PACKAGE) Creating release archives...
	@cd $(DIST_DIR) && \
	for file in $(PROJECT_NAME)-* ; do \
		if [ -f "$$file" ]; then \
			init=""; \
			if [[ "$$file" =~ "server" ]]; then \
				init="-C ../init fcfl-member-sync-server.service"; \
			elif [[ "$$file" =~ "client" ]]; then \
				init="-C ../init fcfl-member-sync-client.service fcfl-member-sync-client.timer"; \
			fi; \
			tar --owner root --group root -czf "$$file.tar.gz" "$$file" $$init || exit 1; \
			rm -f "$$file"; \
		fi \
	done
	$(SUCCESS) All platforms built and archived!

.PHONY: build-debug
build-debug: GCFLAGS += -N -l ## Build with debug symbols
build-debug: BUILD_TAGS += debug
build-debug: build

.PHONY: build-race
build-race: CGO_ENABLED=1 ## Build with race detector
build-race: BUILD_TAGS += race
build-race: build

# =============================================================================
# 📦 Docker Support
# =============================================================================
.PHONY: docker-build
docker-build: ## Build Docker image
	$(WORKING) Building Docker image...
	docker build $(DOCKER_BUILD_ARGS) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-f $(DOCKERFILE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		$(DOCKER_BUILD_CONTEXT)
	$(SUCCESS) Docker image built!

.PHONY: docker-push
docker-push: ## Push Docker image
	$(WORKING) Pushing Docker image...
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	$(SUCCESS) Docker image pushed!

.PHONY: docker-run
docker-run: ## Run Docker container
	$(ROCKET) Running Docker container...
	docker run --rm -it $(DOCKER_IMAGE):$(DOCKER_TAG)

# =============================================================================
# 📊 Reporting & Analytics
# =============================================================================
.PHONY: benchmark-report
benchmark-report:
	@mkdir -p $(DOCS_DIR)/reports
	$(GO) test -bench=. -benchmem ./... > $(DOCS_DIR)/reports/benchmark.txt

.PHONY: lint-report
lint-report:
	@mkdir -p $(DOCS_DIR)/reports
	$(GOLANGCI_LINT) run --out-format checkstyle > $(DOCS_DIR)/reports/lint-checkstyle.xml

.PHONY: security-report
security-report:
	@mkdir -p $(DOCS_DIR)/reports
	$(GOVULNCHECK) -json ./... > $(DOCS_DIR)/reports/security.json

# =============================================================================
# 🧹 Cleanup & Maintenance
# =============================================================================
.PHONY: clean
clean: ## Clean build artifacts
	$(TRASH) Cleaning build artifacts...
	rm -rf $(BIN_DIR) $(DIST_DIR)
	$(SUCCESS) Clean complete!

.PHONY: deps
deps: ## Install dependencies
	$(WORKING) Installing dependencies...
	$(GO) mod download
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@if [ ! -f "$(GOFUMPT)" ]; then \
		$(GO) install mvdan.cc/gofumpt@latest; \
	fi
	@if [ ! -f "$(GOVULNCHECK)" ]; then \
		$(GO) install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@if [ ! -f "$(MOCKGEN)" ]; then \
		$(GO) install github.com/golang/mock/mockgen@latest; \
	fi
	@if [ ! -f "$(AIR)" ]; then \
		$(GO) install github.com/cosmtrek/air@latest; \
	fi
	$(SUCCESS) Dependencies installed!

.PHONY: deps-update
deps-update: ## Update dependencies
	$(WORKING) Updating dependencies...
	$(GO) get -u ./...
	$(GO) mod tidy
	$(SUCCESS) Dependencies updated!

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	$(INFO) Verifying dependencies...
	$(GO) mod verify
	$(SUCCESS) Dependencies verified!

# =============================================================================
# 📚 Documentation
# =============================================================================
.PHONY: docs
docs: $(DOCS_DIR) ## Generate documentation
	$(WORKING) Generating documentation...
	@mkdir -p $(DOCS_DIR)
	$(GO) doc -all > $(DOCS_DIR)/API.md
	@if [ -f "README.md.tmpl" ]; then \
		VERSION=$(VERSION) \
		BUILD_TIME=$(BUILD_TIME) \
		envsubst < README.md.tmpl > README.md; \
	fi
	$(SUCCESS) Documentation generated!

.PHONY: serve-docs
serve-docs: ## Serve documentation locally
	$(ROCKET) Serving documentation at http://localhost:6060
	$(GODOC) -http=:6060

# =============================================================================
# 🛠️ Tools & Utilities
# =============================================================================
.PHONY: tools
tools: ## Install all tools
	$(INFO) Installing tools...
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/golang/mock/mockgen@latest
	$(GO) install github.com/cosmtrek/air@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(SUCCESS) Tools installed!

.PHONY: proto
proto: ## Generate protocol buffers
	$(WORKING) Generating protocol buffers...
	@if [ -d "$(PROTO_DIR)" ]; then \
		protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			$(PROTO_DIR)/*.proto; \
	fi
	$(SUCCESS) Protocol buffers generated!

.PHONY: mock
mock: ## Generate mocks
	$(WORKING) Generating mocks...
	$(MOCKGEN) -source=pkg/interfaces.go -destination=pkg/mocks/mocks.go
	$(SUCCESS) Mocks generated!

.PHONY: version
version: ## Display version information
	@echo -e "$(CYAN)Version:$(RESET)    $(VERSION)"
	@echo -e "$(CYAN)Commit:$(RESET)     $(GIT_COMMIT)"
	@echo -e "$(CYAN)Branch:$(RESET)     $(GIT_BRANCH)"
	@echo -e "$(CYAN)Built:$(RESET)      $(BUILD_TIME)"
	@echo -e "$(CYAN)Built by:$(RESET)   $(BUILT_BY)"
	@echo -e "$(CYAN)Go version:$(RESET) $(shell $(GO) version)"

# =============================================================================
# 📁 Directory Creation
# =============================================================================
$(BIN_DIR) $(DIST_DIR):
	mkdir -p $@

.DEFAULT_GOAL := build
