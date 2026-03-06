BINARY=issuesherpa
GO_SOURCES=$(wildcard *.go) $(wildcard providers/*/*.go) $(wildcard models/*.go)
ENV_FILE=.env

.ONESHELL:

.PHONY: build run tui list open resolved leaderboard offline list-offline clean

build: $(BINARY)

$(BINARY): $(GO_SOURCES)
	go build -o $(BINARY) .

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
