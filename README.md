# IssueSherpa

## Table of Contents

1. [Overview](#1-overview)
2. [Quick Start](#2-quick-start)
3. [Useful Commands](#3-useful-commands)

## 1. Overview

IssueSherpa is a terminal tool for viewing GitHub, GitLab, and Sentry issues in one place.

It solves the context-switching problem of triaging work across separate tools and tabs.

It is also meant to be consumable by AI agents, not just a CLI or TUI.

## 2. Quick Start

### TUI

```bash
cp .env.example .env
# fill in provider tokens/projects in .env
make tui        # builds and runs TUI
./bin/issuesherpa
```

### CLI

```bash
cp .env.example .env
# fill in provider tokens/projects in .env

go run ./cmd/issuesherpa list
go run ./cmd/issuesherpa --offline list
go run ./cmd/issuesherpa leaderboard
```

## 3. Useful Commands

```bash
go run ./cmd/issuesherpa
go build -o bin/issuesherpa ./cmd/issuesherpa
make list
make offline
make test
make check       # includes -race
```

### Runtime notes

- Cache: stored in OS user data dir by default; override with `ISSUESHERPA_DB_PATH=/path/to/issues.db`.
- `--offline` uses cache only and reports staleness.
- `NO_COLOR=1` / `CLICOLOR=0` disable ANSI coloring.
- In TUI, search includes typo-tolerant fallback (for example, `lables` can match `labels`).
