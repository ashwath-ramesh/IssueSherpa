# IssueSherpa

## Table of Contents

1. [Overview](#1-overview)
2. [Install](#2-install)
3. [Quick Start](#3-quick-start)
4. [Commands and Environment](#4-commands-and-environment)
5. [Release](#5-release)

## 1. Overview

IssueSherpa is a terminal tool for viewing GitHub, GitLab, and Sentry issues in one place.

It solves the context-switching problem of triaging work across separate tools and tabs.

It is also meant to be consumable by AI agents, not just a CLI or TUI.

## 2. Install

```bash
curl -fsSL https://raw.githubusercontent.com/ashwath-ramesh/IssueSherpa/master/scripts/install.sh | bash
```

Pin a version:

```bash
curl -fsSL https://raw.githubusercontent.com/ashwath-ramesh/IssueSherpa/master/scripts/install.sh | VERSION=v0.1.0 bash
```

## 3. Quick Start

```bash
issuesherpa init
# edit the generated config.toml
```

### TUI

```bash
issuesherpa
```

### CLI

```bash
issuesherpa list
issuesherpa --offline list
issuesherpa leaderboard
```

## 4. Commands and Environment

```bash
go run ./cmd/issuesherpa
go build -o bin/issuesherpa ./cmd/issuesherpa
make list
make offline
make test
make check       # includes -race
```

- Cache: stored at `XDG_DATA_HOME/issuesherpa/issues.db` or `~/.local/share/issuesherpa/issues.db`; override with `ISSUESHERPA_DB_PATH=/path/to/issues.db`.
- Config: stored at `XDG_CONFIG_HOME/issuesherpa/config.toml` or `~/.config/issuesherpa/config.toml`.
- Existing macOS `~/Library/Application Support/issuesherpa/...` config and DB paths are still honored if present.
- `--offline` uses cache only and reports staleness.
- `NO_COLOR=1` / `CLICOLOR=0` disable ANSI coloring.
- In TUI, search includes typo-tolerant fallback (for example, `lables` can match `labels`).

## 5. Release

See [docs/releasing.md](docs/releasing.md).
