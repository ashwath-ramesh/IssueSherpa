# IssueSherpa

## Quick start

```bash
cp .env.example .env
# fill in provider tokens/projects in .env

make tui
make list
make offline
make test
```

## Notes

- The app stores its cache DB in the OS user data directory by default, not the repo root.
- Override the DB path with `ISSUESHERPA_DB_PATH=/path/to/issues.db`.
- `--offline` reads from cache only and reports when the cache is stale.
- Provider fetch warnings are printed after sync if one source partially fails.

## Direct commands

```bash
go run ./cmd/issuesherpa
go run ./cmd/issuesherpa list
go run ./cmd/issuesherpa --offline
go build -o issuesherpa ./cmd/issuesherpa
./issuesherpa leaderboard
```
