package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sci-ecommerce/issuesherpa/models"
)

type Store struct {
	db *sql.DB
}

type CacheInfo struct {
	LastSyncAt time.Time
	HasSync    bool
	Stale      bool
}

const staleAfter = 24 * time.Hour

func NewStore(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS issues (
		id TEXT PRIMARY KEY,
		short_id TEXT NOT NULL,
		title TEXT NOT NULL,
		status TEXT NOT NULL,
		level TEXT,
		project_id TEXT,
		project_name TEXT,
		project_slug TEXT,
		count TEXT,
		user_count INTEGER,
		first_seen TEXT,
		last_seen TEXT,
		reporter TEXT,
		assigned_to_name TEXT,
		assigned_to_email TEXT,
		source TEXT DEFAULT 'sentry',
		issue_url TEXT
	)`)
	if err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("create metadata table: %w", err)
	}

	for _, column := range []string{"source", "issue_url"} {
		if err := ensureColumn(db, "issues", column); err != nil {
			return nil, fmt.Errorf("migrate table: %w", err)
		}
	}

	return &Store{db: db}, nil
}

func ensureColumn(db *sql.DB, table string, column string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	schema := ""
	switch column {
	case "source":
		schema = "source TEXT DEFAULT 'sentry'"
	case "issue_url":
		schema = "issue_url TEXT"
	}
	if schema == "" {
		return fmt.Errorf("unsupported column migration: %s", column)
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, schema))
	return err
}

func (s *Store) UpsertIssues(issues []models.Issue) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO issues
		(id, short_id, title, status, level,
		 project_id, project_name, project_slug,
		 count, user_count, first_seen, last_seen,
		 reporter, assigned_to_name, assigned_to_email, source, issue_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title,
			status=excluded.status,
			level=excluded.level,
			count=excluded.count,
			user_count=excluded.user_count,
			last_seen=excluded.last_seen,
			reporter=CASE WHEN excluded.reporter != '' THEN excluded.reporter ELSE issues.reporter END,
			assigned_to_name=excluded.assigned_to_name,
			assigned_to_email=excluded.assigned_to_email,
			source=excluded.source,
			issue_url=excluded.issue_url`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, issue := range issues {
		assignedName, assignedEmail := "", ""
		if issue.AssignedTo != nil {
			assignedName = issue.AssignedTo.Name
			assignedEmail = issue.AssignedTo.Email
		}

		source := strings.TrimSpace(issue.Source)
		if source == "" {
			source = "sentry"
		}

		_, err = stmt.Exec(
			issue.ID, issue.ShortID, issue.Title, issue.Status, issue.Level,
			issue.Project.ID, issue.Project.Name, issue.Project.Slug,
			issue.Count, issue.UserCount, issue.FirstSeen, issue.LastSeen,
			issue.Reporter, assignedName, assignedEmail, source, issue.URL,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) LoadIssues() ([]models.Issue, error) {
	rows, err := s.db.Query(`SELECT
		id, short_id, title, status, level,
		project_id, project_name, project_slug,
		count, user_count, first_seen, last_seen,
		reporter, assigned_to_name, assigned_to_email,
		source,
		COALESCE(issue_url,'') AS issue_url
		FROM issues ORDER BY first_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []models.Issue
	for rows.Next() {
		var issue models.Issue
		var assignedName, assignedEmail sql.NullString
		err := rows.Scan(
			&issue.ID, &issue.ShortID, &issue.Title, &issue.Status, &issue.Level,
			&issue.Project.ID, &issue.Project.Name, &issue.Project.Slug,
			&issue.Count, &issue.UserCount, &issue.FirstSeen, &issue.LastSeen,
			&issue.Reporter, &assignedName, &assignedEmail, &issue.Source, &issue.URL,
		)
		if err != nil {
			return nil, err
		}

		issue.Source = normalizeIssueSource(issue.ID, issue.Source)
		if issue.Source == "" {
			issue.Source = "sentry"
		}
		if assignedName.Valid && assignedName.String != "" {
			issue.AssignedTo = &models.AssignedTo{Name: assignedName.String, Email: assignedEmail.String}
		}
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}

func (s *Store) SaveLastSync(at time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO metadata (key, value) VALUES ('last_sync_at', ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		at.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) LoadCacheInfo() (CacheInfo, error) {
	var raw string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = 'last_sync_at'`).Scan(&raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return CacheInfo{}, nil
		}
		return CacheInfo{}, err
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return CacheInfo{}, err
	}

	info := CacheInfo{
		LastSyncAt: parsed,
		HasSync:    true,
		Stale:      time.Since(parsed) > staleAfter,
	}
	return info, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func normalizeIssueSource(id, source string) string {
	if normalized := strings.ToLower(strings.TrimSpace(source)); normalized != "" {
		return normalized
	}
	if strings.HasPrefix(id, "github:") {
		return "github"
	}
	if strings.HasPrefix(id, "gitlab:") {
		return "gitlab"
	}
	if strings.HasPrefix(id, "sentry:") {
		return "sentry"
	}
	return "sentry"
}
