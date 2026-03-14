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

## Direct commands

```bash
go run ./cmd/issuesherpa
go run ./cmd/issuesherpa list
go run ./cmd/issuesherpa --offline
go build -o issuesherpa ./cmd/issuesherpa
./issuesherpa leaderboard
```
