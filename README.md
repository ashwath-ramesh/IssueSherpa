# IssueSherpa

## Table of Contents

1. [Overview](#1-overview)
2. [Quick Start](#2-quick-start)
3. [Notes](#3-notes)
4. [Direct Commands](#4-direct-commands)

## 1. Overview

IssueSherpa is a terminal tool for viewing GitHub, GitLab, and Sentry issues in one place.

It solves the context-switching problem of triaging work across separate tools and tabs.

## 2. Quick Start

```bash
cp .env.example .env
# fill in provider tokens/projects in .env

make tui
make list
make offline
make test
```

## 3. Notes

- The app stores its cache DB in the OS user data directory by default, not the repo root.
- Override the DB path with `ISSUESHERPA_DB_PATH=/path/to/issues.db`.
- `--offline` reads from cache only and reports when the cache is stale.
- Provider fetch warnings are printed after sync if one source partially fails.

## 4. Direct Commands

```bash
go run ./cmd/issuesherpa
go run ./cmd/issuesherpa list
go run ./cmd/issuesherpa --offline
go build -o issuesherpa ./cmd/issuesherpa
./issuesherpa leaderboard
```
