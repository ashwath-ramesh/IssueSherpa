BINARY_DIR=bin
BINARY=$(BINARY_DIR)/issuesherpa
GO_SOURCES=$(shell find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*')
GO_MODULE_FILES=go.mod go.sum
ENV_FILE=.env
OS_NAME:=$(shell uname -s)
ifeq ($(OS_NAME),Darwin)
DEFAULT_DB_PATH:=$(HOME)/Library/Application Support/issuesherpa/issues.db
else ifeq ($(OS_NAME),Linux)
DEFAULT_DB_PATH:=$(if $(XDG_DATA_HOME),$(XDG_DATA_HOME),$(HOME)/.local/share)/issuesherpa/issues.db
else
DEFAULT_DB_PATH:=issues.db
endif
DB_PATH?=$(if $(ISSUESHERPA_DB_PATH),$(ISSUESHERPA_DB_PATH),$(DEFAULT_DB_PATH))

.ONESHELL:

.PHONY: build test test-race check run tui list open resolved leaderboard offline list-offline clean

build: $(BINARY)

$(BINARY): $(GO_SOURCES) $(GO_MODULE_FILES)
	mkdir -p $(BINARY_DIR)
	go build -o $(BINARY) ./cmd/issuesherpa

test:
	go test ./...

test-race:
	go test -race ./...

check: test-race

define run_with_env
	@if [ -f $(ENV_FILE) ]; then \
		set -a; \
		. ./$(ENV_FILE); \
		set +a; \
	fi; \
	$(1)
endef

tui: build
	$(call run_with_env,$(BINARY))

run: tui

list: build
	$(call run_with_env,$(BINARY) list)

open: build
	$(call run_with_env,$(BINARY) list --open)

resolved: build
	$(call run_with_env,$(BINARY) list --resolved)

leaderboard: build
	$(call run_with_env,$(BINARY) leaderboard)

offline: build
	$(call run_with_env,$(BINARY) --offline)

list-offline: build
	$(call run_with_env,$(BINARY) --offline list)

clean:
	rm -f issuesherpa issuesherpa.exe
	rm -f $(BINARY)
	rmdir $(BINARY_DIR) 2>/dev/null || true
	rm -f $(DB_PATH)
