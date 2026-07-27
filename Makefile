BINARY := clione
BIN_DIR := bin
COMPLETIONS_DIR := completions

# Version stamped into the binary (shown by `clione --version` and in the TUI
# header). Same mechanism CI/release use, so local builds reflect reality
# instead of always saying "dev".
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

# Shell completion install locations. Override on the command line if your
# setup differs, e.g. `make install-completions ZSH_COMPLETION_DIR=/usr/local/share/zsh/site-functions`.
BASH_COMPLETION_DIR ?= $(HOME)/.local/share/bash-completion/completions
ZSH_COMPLETION_DIR  ?= $(HOME)/.local/share/zsh/site-functions
FISH_COMPLETION_DIR ?= $(HOME)/.config/fish/completions

.PHONY: build test lint vet run clean completions install-completions

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/clione

test:
	go test ./...

vet:
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; running go vet instead"; \
		go vet ./...; \
	fi

run:
	go run ./cmd/clione

completions:
	@echo "Shell completion scripts live in $(COMPLETIONS_DIR)/:"
	@ls -1 $(COMPLETIONS_DIR)
	@echo "Run 'make install-completions' to install them for bash, zsh, and fish."

install-completions:
	@install -d "$(BASH_COMPLETION_DIR)" "$(ZSH_COMPLETION_DIR)" "$(FISH_COMPLETION_DIR)"
	@install -m 0644 "$(COMPLETIONS_DIR)/clione.bash" "$(BASH_COMPLETION_DIR)/clione"
	@install -m 0644 "$(COMPLETIONS_DIR)/clione.zsh"  "$(ZSH_COMPLETION_DIR)/_clione"
	@install -m 0644 "$(COMPLETIONS_DIR)/clione.fish" "$(FISH_COMPLETION_DIR)/clione.fish"
	@echo "Installed clione completions:"
	@echo "  bash -> $(BASH_COMPLETION_DIR)/clione"
	@echo "  zsh  -> $(ZSH_COMPLETION_DIR)/_clione (ensure this dir is in your fpath)"
	@echo "  fish -> $(FISH_COMPLETION_DIR)/clione.fish"
	@echo "Restart your shell (or re-source your rc) to activate them."

clean:
	rm -rf $(BIN_DIR) dist coverage.out
