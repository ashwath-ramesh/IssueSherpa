BINARY=issuesherpa
GO_SOURCES=$(shell find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*')
GO_MODULE_FILES=go.mod go.sum
ENV_FILE=.env

.ONESHELL:

.PHONY: build test run tui list open resolved leaderboard offline list-offline clean

build: $(BINARY)

$(BINARY): $(GO_SOURCES) $(GO_MODULE_FILES)
	go build -o $(BINARY) ./cmd/issuesherpa

test:
	go test ./...

define run_with_env
	@if [ -f $(ENV_FILE) ]; then \
		set -a; \
		. ./$(ENV_FILE); \
		set +a; \
	fi; \
	$(1)
endef

tui: build
	$(call run_with_env,./$(BINARY))

run: tui

list: build
	$(call run_with_env,./$(BINARY) list)

open: build
	$(call run_with_env,./$(BINARY) list --open)

resolved: build
	$(call run_with_env,./$(BINARY) list --resolved)

leaderboard: build
	$(call run_with_env,./$(BINARY) leaderboard)

offline: build
	$(call run_with_env,./$(BINARY) --offline)

list-offline: build
	$(call run_with_env,./$(BINARY) --offline list)

clean:
	rm -f $(BINARY) issues.db
