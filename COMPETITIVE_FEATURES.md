# IssueSherpa: Competitive Feature Analysis

Consolidated from 10 open source projects. Ranked by estimated developer usage (how many devs actively use this feature daily). Each feature includes how IssueSherpa can extend it to our multi-source (GitHub + GitLab + Sentry) use case.

---

## Tier 1: Table Stakes (Every Dev Uses Daily)

### 1. Configurable Filter Sections with Saved Views
**Usage: ~95% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [jira-cli](https://github.com/ankitpokhrel/jira-cli)

YAML-defined sections, each with its own query. gh-dash lets you define unlimited tabs like "My PRs", "Needs Review", "Bugs". jira-cli adds composable negation (`-s~Done` = not done) and time-relative filters (`--created -7d`).

**IssueSherpa extension:** Sections that span sources. A single "Critical" tab showing Sentry errors with >100 events + GitHub issues labeled `priority:high` + GitLab issues marked `critical`. No other tool can filter across Sentry error counts AND issue tracker labels in one view.

### 2. Vim-Style Keyboard Navigation + Detail Sidebar
**Usage: ~90% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [taskwarrior-tui](https://github.com/kdheepak/taskwarrior-tui), [jira-cli](https://github.com/ankitpokhrel/jira-cli)

`j/k` movement, `Enter` to expand, tabbed detail pane (overview/checks/files/commits in gh-dash), zoom toggle (`z` in taskwarrior-tui) for inline detail expansion.

**IssueSherpa extension:** Detail sidebar that correlates across sources. Select a Sentry error and see linked GitHub issues, or select a GitHub issue and see related Sentry errors (matched by title/tag). The sidebar becomes a cross-reference tool, not just a detail viewer.

### 3. Offline-First with Local Cache
**Usage: ~80% of users** | Source: [git-bug](https://github.com/git-bug/git-bug), [super-productivity](https://github.com/johannesjo/super-productivity)

git-bug stores issues as git objects for full offline editing. Super Productivity keeps everything local with optional Dropbox/WebDAV sync. IssueSherpa already has SQLite caching.

**IssueSherpa extension:** Already implemented. Extend with cache freshness indicators (show "synced 3h ago" per source) and selective sync (`issuesherpa sync --source sentry` to refresh only Sentry data on slow connections).

### 4. Open-in-Browser / URL Copy
**Usage: ~80% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [jira-cli](https://github.com/ankitpokhrel/jira-cli)

`Enter` opens issue in browser, `c` copies URL. Simple but used constantly.

**IssueSherpa extension:** Already have URL field. Add `o` to open in browser, `y` to yank URL. For Sentry issues, open directly to the error detail page, not just the project page.

---

## Tier 2: Power User Features (Daily for ~50-70% of devs)

### 5. Cross-Source Issue Normalization + Priority Mapping
**Usage: ~70% of users** | Source: [bugwarrior](https://github.com/GothenburgBitFactory/bugwarrior)

bugwarrior maps every source's priority scheme to a unified H/M/L scale (e.g., Jira's "Blocker" -> H, GitHub label "p1" -> H). Uses User Defined Attributes (UDAs) to preserve source-specific fields alongside normalized ones.

**IssueSherpa extension:** Map Sentry `level` (fatal/error/warning/info) + event count to a unified priority score. A Sentry fatal with 500 events should rank higher than a GitHub issue labeled "bug". Define a scoring formula: `priority = f(source_severity, event_count, user_count, age)`. This is the killer feature no tool has.

### 6. Custom Keybindings with Shell Command Execution
**Usage: ~60% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [taskwarrior-tui](https://github.com/kdheepak/taskwarrior-tui)

gh-dash: bind any key to shell commands with template variables (`gh pr merge --repo {{.RepoName}} {{.PrNumber}}`). taskwarrior-tui: keys 1-9 invoke user scripts, passing selected item UUIDs.

**IssueSherpa extension:** Bind keys to cross-source workflows. Example: press `f` on a Sentry error to auto-create a GitHub issue pre-filled with the error title, stacktrace link, and event count. Press `r` to resolve a Sentry issue AND close the linked GitHub issue in one keystroke.

### 7. Live Filter with Readline Editing
**Usage: ~60% of users** | Source: [taskwarrior-tui](https://github.com/kdheepak/taskwarrior-tui)

Full Emacs-style line editing (`Ctrl+a/e/k/u/w`, `Alt+b/f`) in the filter prompt. Typing immediately updates the list -- no submit step. Tab completion for filter values.

**IssueSherpa extension:** Filter across all sources in real-time. Type `source:sentry level:fatal` or `project:myapp status:open` and see results instantly filtered across GitHub, GitLab, and Sentry. Autocomplete source names, project slugs, and label values from cached data.

### 8. Leaderboard / Assignment Analytics
**Usage: ~50% of users** | Source: IssueSherpa (already built), [plane](https://github.com/makeplane/plane)

IssueSherpa already has `issuesherpa leaderboard`. Plane adds workload distribution dashboards and cycle velocity charts.

**IssueSherpa extension:** Already ahead here. Extend leaderboard to show cross-source load: "Alice: 12 GitHub issues, 3 Sentry errors assigned, 2 GitLab MRs pending review". Add `--team` flag to aggregate by team. Show who's overloaded across all platforms.

---

## Tier 3: Differentiating Features (Weekly use, high retention value)

### 9. Triage Inbox (Unified Incoming Queue)
**Usage: ~45% of users** | Source: [tegon](https://github.com/tegonhq/tegon), [plane](https://github.com/makeplane/plane)

Tegon's killer feature: a dedicated inbox aggregating issues from Slack, GitHub, email, and monitoring into a scan-assess-decide queue. Accept (assign + set status) or decline. Bidirectional sync keeps Slack threads updated.

**IssueSherpa extension:** A `triage` mode showing only new/unseen issues across all sources since last sync. Sentry errors that spiked overnight + new GitHub issues + new GitLab issues in one queue. Mark as "acknowledged", "snoozed", or "create-issue" (for Sentry errors that need a tracking issue). This is the daily-standup killer feature.

### 10. AI-Powered Issue Enrichment
**Usage: ~40% of users (growing fast)** | Source: [tegon](https://github.com/tegonhq/tegon), [plane](https://github.com/makeplane/plane)

Tegon's Bug Enricher: when a bug is created, an LLM auto-generates a multi-step resolution guide with code snippets, posted as a comment. Plane has AI agents for triaging and assigning.

**IssueSherpa extension:** `issuesherpa explain <id>` sends the Sentry stacktrace + linked GitHub issue description to an LLM and returns a root-cause hypothesis. For cross-source correlation: "This Sentry error started spiking after PR #142 was merged 2 days ago" by matching Sentry `firstSeen` timestamps against recent GitHub/GitLab merge events.

### 11. Bi-Directional Bridges (Sync Back to Source)
**Usage: ~40% of users** | Source: [git-bug](https://github.com/git-bug/git-bug), [bugwarrior](https://github.com/GothenburgBitFactory/bugwarrior)

git-bug: edit issues locally, `git bug push` syncs changes back to GitHub/GitLab/Jira. Incremental, resumable sync with conflict resolution via CRDTs.

**IssueSherpa extension:** Write-back actions from the TUI. Resolve a Sentry issue, close a GitHub issue, change a GitLab issue's label -- all without leaving the terminal. Start read-only (current), add write-back as a v2 feature.

### 12. Multi-Select with Bulk Operations
**Usage: ~40% of users** | Source: [taskwarrior-tui](https://github.com/kdheepak/taskwarrior-tui), [gh-dash](https://github.com/dlvhdr/gh-dash)

`v` marks individual items, `V` marks all visible, then batch-apply actions (close, label, assign, snooze).

**IssueSherpa extension:** Bulk-resolve Sentry errors that were fixed by the same deploy. Bulk-close GitHub issues from a completed milestone. Bulk-assign unassigned issues across sources during triage.

### 13. Auto-Scoping to Current Repo
**Usage: ~40% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash)

When launched from a git clone directory, gh-dash auto-detects the remote and filters to that repo. Toggle with `t`.

**IssueSherpa extension:** Detect the current repo's origin URL, auto-filter to matching GitHub/GitLab project AND matching Sentry project (via config mapping). `cd myapp && issuesherpa` shows only myapp's issues + errors. Zero-config for the common case.

---

## Tier 4: Nice-to-Have (Monthly use, retention boosters)

### 14. Dashboard Layout with Multi-Widget Tiling
**Usage: ~30% of users** | Source: [wtfutil](https://github.com/wtfutil/wtf)

YAML-defined grid layout with coordinate-based widget placement. Any shell command becomes a widget via CmdRunner.

**IssueSherpa extension:** Split-pane dashboard mode: left pane = Sentry errors sorted by frequency, right pane = GitHub/GitLab issues sorted by priority, bottom pane = recent activity feed. Configurable via YAML. Think `tmux` but for issue triage.

### 15. Multiple Output Formats (JSON, CSV, Plain)
**Usage: ~30% of users** | Source: [jira-cli](https://github.com/ankitpokhrel/jira-cli)

`--plain` for scripting, `--raw` for JSON, CSV export, column selection with `--columns`.

**IssueSherpa extension:** `issuesherpa list --json` for piping to `jq`, `--csv` for spreadsheet export, `--columns source,title,priority,count` for custom output. Enable shell scripting: `issuesherpa list --json | jq '.[] | select(.source=="sentry" and .count > 100)'`.

### 16. Notification / Change Detection
**Usage: ~25% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [super-productivity](https://github.com/johannesjo/super-productivity)

gh-dash: full GitHub notification inbox in the TUI with read/unread/done states and bookmarks. Super Productivity: real-time alerts when upstream issues change.

**IssueSherpa extension:** Diff between syncs. `issuesherpa sync` shows "12 new Sentry errors, 3 new GitHub issues, 2 issues closed since last sync". Desktop notification when a Sentry error crosses a threshold (e.g., >1000 events).

### 17. Per-Project Config Overrides
**Usage: ~25% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [jira-cli](https://github.com/ankitpokhrel/jira-cli)

gh-dash: `.gh-dash.yml` in repo root for project-specific sections. jira-cli: `--config` flag for profile switching. bugwarrior: named "flavors" (`--flavor work`).

**IssueSherpa extension:** `.issuesherpa.yml` in repo root mapping this project to its Sentry project slug and GitHub/GitLab repo. Team-level config in `~/.config/issuesherpa/config.yml`. Switch profiles with `--profile work|personal|oss`.

### 18. Time-Relative Dynamic Filters
**Usage: ~20% of users** | Source: [gh-dash](https://github.com/dlvhdr/gh-dash), [jira-cli](https://github.com/ankitpokhrel/jira-cli)

gh-dash: Go templates in filter strings (`updated:>={{ nowModify "-3w" }}`). jira-cli: natural language dates (`--created -7d`, `--updated month`).

**IssueSherpa extension:** `issuesherpa list --since 24h` shows issues/errors from the last day across all sources. In TUI: a "Last 24h" / "Last 7d" / "This sprint" quick-filter toggle. Critical for daily standups and on-call handoffs.

### 19. Tagging System with Label Import
**Usage: ~20% of users** | Source: [bugwarrior](https://github.com/GothenburgBitFactory/bugwarrior)

bugwarrior converts source labels to local tags via Jinja2 templates (`github_{{label}}`). Supports add/remove semantics and sprint-to-tag conversion.

**IssueSherpa extension:** Normalize labels across sources. GitHub `bug` + GitLab `type::bug` + Sentry `level:error` all map to a unified `bug` tag. Enable cross-source filtering by semantic category rather than source-specific label names.

### 20. GraphQL API / MCP Server
**Usage: ~15% of users** | Source: [git-bug](https://github.com/git-bug/git-bug), [plane](https://github.com/makeplane/plane)

git-bug exposes a full GraphQL API with subscriptions for live updates. Plane ships an MCP server so AI agents can interact with it programmatically.

**IssueSherpa extension:** Expose a local GraphQL or REST API so other tools (Raycast, Alfred, Claude, custom scripts) can query the unified issue cache. Ship an MCP server so AI coding assistants can ask "what are the top Sentry errors this week?" directly.

---

## Summary: Implementation Priority Matrix

| Priority | Feature | Effort | Impact |
|----------|---------|--------|--------|
| **P0** | Saved filter sections (YAML) | Medium | Table stakes for TUI tools |
| **P0** | Vim keys + detail sidebar | Medium | Already partially built |
| **P0** | Open-in-browser / copy URL | Small | Quick win |
| **P1** | Cross-source priority scoring | Medium | **Unique differentiator** |
| **P1** | Triage inbox mode | Medium | **Unique differentiator** |
| **P1** | Auto-scope to current repo | Small | Huge UX improvement |
| **P1** | Custom keybindings + shell commands | Medium | Power user retention |
| **P2** | AI issue enrichment | Medium | Trending, high wow-factor |
| **P2** | Live filter with readline | Medium | Power user daily driver |
| **P2** | Multi-select + bulk ops | Medium | Triage workflow essential |
| **P2** | JSON/CSV output formats | Small | Scripting ecosystem |
| **P3** | Bi-directional write-back | Large | v2 feature |
| **P3** | Dashboard tiling layout | Large | Nice to have |
| **P3** | MCP server / API | Medium | AI-native future-proofing |
| **P3** | Sync diff notifications | Medium | On-call use case |
