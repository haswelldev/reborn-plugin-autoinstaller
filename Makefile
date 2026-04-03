## Reborn Plugin Autoinstaller — Makefile
## Cross-compiles from macOS to Windows x64.
##
## One-time setup:
##   brew install go mingw-w64 nsis
##   go install github.com/akavel/rsrc@latest

export PATH := /opt/homebrew/bin:$(PATH)

VERSION    := 1.0.0
DIST       := dist
APP_EXE    := $(DIST)/reborn-plugin-autoinstaller.exe
INSTALLER  := $(DIST)/RebornPluginAutoinstaller-Setup.exe
RSRC_BIN   := $(shell go env GOPATH)/bin/rsrc

CC         := x86_64-w64-mingw32-gcc
GOOS       := windows
GOARCH     := amd64
CGO_ENABLED := 1
LDFLAGS    := -H windowsgui -s -w

.PHONY: all build rsrc installer clean help

all: build ## Default: build the Windows exe

build: rsrc $(DIST) ## Build the Windows exe into dist/
	CC=$(CC) GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -ldflags="$(LDFLAGS)" -o $(APP_EXE) .
	@echo "Built: $(APP_EXE) ($$(du -sh $(APP_EXE) | cut -f1))"

rsrc: ## Embed app.manifest + icon into rsrc.syso (required before build)
	$(RSRC_BIN) -manifest app.manifest -ico resources/icon.ico -o rsrc.syso
	@echo "Generated rsrc.syso"

installer: build ## Build NSIS installer into dist/ (requires Linux or Windows — see README)
	@if command -v makensis >/dev/null 2>&1; then \
		makensis installer.nsi && echo "Built: $(INSTALLER)"; \
	else \
		echo ""; \
		echo "makensis not found or not working on this platform."; \
		echo "To build the installer:"; \
		echo "  - On Linux:   sudo apt install nsis && make installer"; \
		echo "  - On Windows: run build-installer.bat"; \
		echo "  - Via CI:     push a tag — GitHub Actions builds it automatically"; \
		echo ""; \
	fi

clean: ## Remove all build artifacts
	rm -rf $(DIST)/ rsrc.syso
	@echo "Cleaned."

$(DIST):
	mkdir -p $(DIST)

help: ## Show available targets
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo ""
