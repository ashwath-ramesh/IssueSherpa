package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
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

	for _, col := range []string{
		"source",
		"issue_url",
	} {
		if err := ensureColumn(db, "issues", col); err != nil {
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

	schema := ""
	switch column {
	case "source":
		schema = "source TEXT DEFAULT 'sentry'"
	case "issue_url":
		schema = "issue_url TEXT"
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, schema))
	return err
}

func (s *Store) UpsertIssues(issues []Issue) error {
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

	for _, i := range issues {
		aName, aEmail := "", ""
		if i.AssignedTo != nil {
			aName = i.AssignedTo.Name
			aEmail = i.AssignedTo.Email
		}
		source := strings.TrimSpace(i.Source)
		if source == "" {
			source = "sentry"
		}
		_, err = stmt.Exec(
			i.ID, i.ShortID, i.Title, i.Status, i.Level,
			i.Project.ID, i.Project.Name, i.Project.Slug,
			i.Count, i.UserCount, i.FirstSeen, i.LastSeen,
			i.Reporter, aName, aEmail, source, i.URL,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) LoadIssues() ([]Issue, error) {
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

	var issues []Issue
	for rows.Next() {
		var i Issue
		var aName, aEmail sql.NullString
		err := rows.Scan(
			&i.ID, &i.ShortID, &i.Title, &i.Status, &i.Level,
			&i.Project.ID, &i.Project.Name, &i.Project.Slug,
			&i.Count, &i.UserCount, &i.FirstSeen, &i.LastSeen,
			&i.Reporter, &aName, &aEmail, &i.Source, &i.URL,
		)
		if err != nil {
			return nil, err
		}
			i.Source = normalizeIssueSource(i.ID, i.Source)
			if i.Source == "" {
				i.Source = "sentry"
			}
		if aName.Valid && aName.String != "" {
			i.AssignedTo = &AssignedTo{Name: aName.String, Email: aEmail.String}
		}
		issues = append(issues, i)
	}

	return issues, nil
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

func (s *Store) ReporterCache() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT id, reporter FROM issues WHERE reporter != '' AND reporter IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cache := map[string]string{}
	for rows.Next() {
		var id, reporter string
		if err := rows.Scan(&id, &reporter); err != nil {
			return nil, err
		}
		cache[id] = reporter
	}
	return cache, nil
}

func (s *Store) HasData() bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM issues").Scan(&count)
	return count > 0
}

func (s *Store) Close() {
	s.db.Close()
}
