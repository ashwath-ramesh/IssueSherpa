# IssueSherpa

## Table of Contents

1. [Overview](#1-overview)
2. [Quick Start](#2-quick-start)
3. [Notes](#3-notes)
4. [Direct Commands](#4-direct-commands)

## 1. Overview

IssueSherpa is a terminal tool for viewing GitHub, GitLab, and Sentry issues in one place.

It solves the context-switching problem of triaging work across separate tools and tabs.

It is also meant to be consumable by AI agents, not just a CLI or TUI.

## 2. Quick Start

```bash
cp .env.example .env
# fill in provider tokens/projects in .env

make tui
./bin/issuesherpa
make list
make offline
make test
make test-race
```

## 3. Notes

- The app stores its cache DB in the OS user data directory by default, not the repo root.
- Override the DB path with `ISSUESHERPA_DB_PATH=/path/to/issues.db`.
- `--offline` reads from cache only and reports when the cache is stale.
- Provider fetch warnings are printed after sync if one source partially fails.
- Search is tolerant in TUI (`/`): exact matches first, then small typo-tolerant fallback on ID/title/project/reporter/source.
- Example: searching `lables` can still match `labels`.
- In-TUI refresh is available with `Ctrl+R` and shows a live refresh status line in the header.

### TUI controls

- `j/k`, `pgup/pgdown`: move selection
- `1/2/3`: open/all/resolved scope
- `p`: project filter, `v`: provider filter, `s`: sort, `r`: reverse sort
- `/`: search mode (fuzzy matching active), `ctrl+r`: refresh while in TUI, `q`: quit

## 4. Direct Commands

```bash
go run ./cmd/issuesherpa
go run ./cmd/issuesherpa list
go run ./cmd/issuesherpa --offline
go build -o bin/issuesherpa ./cmd/issuesherpa
./bin/issuesherpa leaderboard
```

### Terminal behavior

```bash
NO_COLOR=1 ./bin/issuesherpa  # force plain output in terminals without color
CLICOLOR=0 ./bin/issuesherpa

# run tests:
make test
make check       # includes -race
```
