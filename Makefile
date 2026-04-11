.PHONY: build build-linux-amd64 package-deb package-rpm install-local clean test lint fmt vet mod-tidy help

VERSION ?= 1.0.0.1
RELEASE ?= $(shell git rev-parse --short=8 HEAD)
COMMIT ?= $(RELEASE)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BINARY := keydesk
PROJECT_NAME_LOWERCASE := keydesk
BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
INSTALL_DIR := /usr
CONFIG_DIR := /etc
SHARE_INSTALL_DESTINATION := /usr/share/$(PROJECT_NAME_LOWERCASE)
RUN_DIR_PATH := /var/run/$(PROJECT_NAME_LOWERCASE)
PIDFILE_PATH := $(RUN_DIR_PATH)/$(PROJECT_NAME_LOWERCASE).pid
CONFIG_PATH := /etc/$(PROJECT_NAME_LOWERCASE).conf

GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=makefile"
GCFLAGS := -gcflags="all=-trimpath=$(PWD)"
ASMFLAGS := -asmflags="all=-trimpath=$(PWD)"

all: build

generate-version:
	@echo "Generating version.go..."
	PROJECT_NAME_LOWERCASE=$(PROJECT_NAME_LOWERCASE) \
	PROJECT_VERSION=$(VERSION) \
	PROJECT_VERSION_GIT=$(COMMIT) \
	SHARE_INSTALL_DESTINATION=$(SHARE_INSTALL_DESTINATION) \
	RUN_DIR_PATH=$(RUN_DIR_PATH) \
	PIDFILE_PATH=$(PIDFILE_PATH) \
	CONFIG_PATH=$(CONFIG_PATH) \
	OUTPUT_FILE=src/app/version/version.go \
	./scripts/generate_version.sh

frontend-install:
	@command -v npm >/dev/null 2>&1 || { echo "ERROR: npm is required for frontend build. Install Node.js first."; exit 1; }
	@cd src/frontend && npm install --silent

frontend-build: frontend-install
	@echo "Building frontend..."
	@cd src/frontend && npm run build

frontend-check: frontend-install
	@echo "Type-checking frontend..."
	@cd src/frontend && npm run check

frontend-watch:
	@cd src/frontend && npm run watch

build: generate-version frontend-build
	@echo "Building $(BINARY) $(VERSION) for current platform..."
	@mkdir -p $(BIN_DIR)
	cd src && $(GOBUILD) $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o ../$(BIN_DIR)/$(BINARY) ./cmd/keydesk.go

build-linux-amd64: generate-version frontend-build
	@echo "Building $(BINARY) $(VERSION) for linux/amd64..."
	@mkdir -p $(BIN_DIR)
	cd src && GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o ../$(BIN_DIR)/$(BINARY)-linux-amd64 ./cmd/keydesk.go

build-all: build-linux-amd64

package-deb-amd64: build-linux-amd64
	@echo "Creating DEB package for amd64..."
	@mkdir -p $(BUILD_DIR)
	@cp "$(BIN_DIR)/$(BINARY)-linux-amd64" "$(BIN_DIR)/$(BINARY)"
	@VERSION=$(VERSION) RELEASE=$(RELEASE) ARCH=amd64 PATH=$$PATH:$(shell go env GOPATH)/bin nfpm pkg --config nfpm.yaml --packager deb --target $(BUILD_DIR)/
	@rm "$(BIN_DIR)/$(BINARY)"
	@if [ -f "$(BUILD_DIR)/$(BINARY)_$(VERSION)-$(RELEASE)_amd64.deb" ]; then \
		mv "$(BUILD_DIR)/$(BINARY)_$(VERSION)-$(RELEASE)_amd64.deb" "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE)-amd64.deb"; \
	fi

package-rpm-amd64: build-linux-amd64
	@echo "Creating RPM package for amd64..."
	@mkdir -p $(BUILD_DIR)
	@cp "$(BIN_DIR)/$(BINARY)-linux-amd64" "$(BIN_DIR)/$(BINARY)"
	@VERSION=$(VERSION) RELEASE=$(RELEASE) ARCH=amd64 PATH=$$PATH:$(shell go env GOPATH)/bin nfpm pkg --config nfpm.yaml --packager rpm --target $(BUILD_DIR)/
	@rm "$(BIN_DIR)/$(BINARY)"
	@if [ -f "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE).x86_64.rpm" ]; then \
		mv "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE).x86_64.rpm" "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE)-x86_64.rpm"; \
	fi

package-deb: build-all
	@echo "Creating DEB packages for all architectures..."
	@for arch in amd64; do \
		if [ -f "$(BIN_DIR)/$(BINARY)-linux-$$arch" ]; then \
			echo "Creating DEB package for $$arch..."; \
			cp "$(BIN_DIR)/$(BINARY)-linux-$$arch" "$(BIN_DIR)/$(BINARY)"; \
			VERSION=$(VERSION) RELEASE=$(RELEASE) ARCH=$$arch PATH=$$PATH:$(shell go env GOPATH)/bin nfpm pkg --config nfpm.yaml --packager deb --target $(BUILD_DIR)/; \
			rm "$(BIN_DIR)/$(BINARY)"; \
		fi; \
	done
	@if [ -f "$(BUILD_DIR)/$(BINARY)_$(VERSION)-$(RELEASE)_amd64.deb" ]; then \
		mv "$(BUILD_DIR)/$(BINARY)_$(VERSION)-$(RELEASE)_amd64.deb" "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE)-amd64.deb"; \
	fi

package-rpm: build-all
	@echo "Creating RPM packages for all architectures..."
	@for arch in amd64; do \
		if [ -f "$(BIN_DIR)/$(BINARY)-linux-$$arch" ]; then \
			echo "Creating RPM package for $$arch..."; \
			rpm_arch=$$arch; \
			if [ "$$arch" = "amd64" ]; then rpm_arch="x86_64"; fi; \
			cp "$(BIN_DIR)/$(BINARY)-linux-$$arch" "$(BIN_DIR)/$(BINARY)"; \
			VERSION=$(VERSION) RELEASE=$(RELEASE) ARCH=$$rpm_arch PATH=$$PATH:$(shell go env GOPATH)/bin nfpm pkg --config nfpm.yaml --packager rpm --target $(BUILD_DIR)/; \
			rm "$(BIN_DIR)/$(BINARY)"; \
		fi; \
	done
	@if [ -f "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE).x86_64.rpm" ]; then \
		mv "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE).x86_64.rpm" "$(BUILD_DIR)/$(BINARY)-$(VERSION)-$(RELEASE)-x86_64.rpm"; \
	fi

package-all: package-deb package-rpm

install-local: build
	@echo "Installing locally..."
	@sudo mkdir -p $(INSTALL_DIR)/bin
	@sudo mkdir -p $(SHARE_INSTALL_DESTINATION)
	@sudo mkdir -p /usr/share/doc/$(PROJECT_NAME_LOWERCASE)
	@sudo install -m 755 $(BIN_DIR)/$(BINARY) /usr/bin/$(BINARY)
	@sudo cp -r src/install/* $(SHARE_INSTALL_DESTINATION)/
	@sudo install -m 644 config/$(BINARY).conf $(CONFIG_DIR)/$(BINARY).conf
	@sudo install -m 644 LICENSE /usr/share/doc/$(PROJECT_NAME_LOWERCASE)/LICENSE

test:
	cd src && $(GOTEST) -v ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	cd src && golangci-lint run

fmt:
	cd src && $(GOFMT) -s -w .

vet:
	cd src && $(GOVET) ./...

mod-tidy:
	cd src && $(GOMOD) tidy

mod-download:
	cd src && $(GOMOD) download

clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

dev-setup: mod-download frontend-install
	@echo "Development environment setup complete"

ci: mod-tidy fmt vet test frontend-check frontend-build build-all

help:
	@echo "Available targets:"
	@echo "  build              - Build for current platform"
	@echo "  build-linux-amd64  - Build for Linux AMD64"
	@echo "  build-all          - Build for all platforms"
	@echo "  package-deb-amd64  - Create DEB package (amd64)"
	@echo "  package-rpm-amd64  - Create RPM package (amd64)"
	@echo "  package-all        - Create all packages"
	@echo "  install-local      - Install locally for development"
	@echo "  test               - Run tests"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  vet                - Run go vet"
	@echo "  mod-tidy           - Tidy go modules"
	@echo "  clean              - Clean build artifacts"
	@echo "  frontend-build     - Build frontend TypeScript"
	@echo "  frontend-check     - Type-check frontend"
	@echo "  frontend-watch     - Watch frontend for changes"
	@echo "  dev-setup          - Setup development environment"
	@echo "  ci                 - Run CI pipeline locally"
	@echo "  help               - Show this help"
