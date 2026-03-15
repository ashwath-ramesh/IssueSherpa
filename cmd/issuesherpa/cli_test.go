package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestRunCLIDefaultJSONOutput(t *testing.T) {
	issues := sampleTestIssues()
	out, stderr, err := runCLICaptureOutput(t, []string{"list"}, issues)
	if err != nil {
		t.Fatalf("runCLI list returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}

	var payload struct {
		Command string                   `json:"command"`
		Count   int                      `json:"count"`
		Total   int                      `json:"total"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.Command != "list" {
		t.Fatalf("command = %s, want list", payload.Command)
	}
	if payload.Count != len(issues) {
		t.Fatalf("count = %d, want %d", payload.Count, len(issues))
	}
	if payload.Total != len(issues) {
		t.Fatalf("total = %d, want %d", payload.Total, len(issues))
	}
	if len(payload.Data) != len(issues) {
		t.Fatalf("data len = %d, want %d", len(payload.Data), len(issues))
	}
}

func TestRunCLITextOutput(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"list", "--output", "text"}, issues)
	if err != nil {
		t.Fatalf("runCLI text list returned error: %v", err)
	}
	if !strings.Contains(out, "SOURCE") {
		t.Fatalf("expected text table header, got %q", out)
	}
	if !strings.Contains(out, "Total:") {
		t.Fatalf("expected total line in text output, got %q", out)
	}
}

func TestRunCLIOutputNDJSON(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"list", "--output", "ndjson"}, issues)
	if err != nil {
		t.Fatalf("runCLI ndjson list returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != len(issues) {
		t.Fatalf("ndjson lines = %d, want %d", len(lines), len(issues))
	}
	for _, line := range lines {
		var payload struct {
			Command string                 `json:"command"`
			Count   int                    `json:"count"`
			Total   int                    `json:"total"`
			Data    map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("invalid ndjson line: %v", err)
		}
		if payload.Command != "list" {
			t.Fatalf("command = %s, want list", payload.Command)
		}
		if payload.Count != 1 {
			t.Fatalf("count = %d, want 1", payload.Count)
		}
		if payload.Total != len(issues) {
			t.Fatalf("total = %d, want %d", payload.Total, len(issues))
		}
		if _, ok := payload.Data["id"]; !ok {
			t.Fatalf("missing issue data in ndjson line: %q", line)
		}
	}
}

func TestRunCLISchema(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"--schema"}, sampleTestIssues())
	if err != nil {
		t.Fatalf("runCLI --schema returned error: %v", err)
	}
	var payload struct {
		Version  string `json:"version"`
		Name     string `json:"name"`
		Commands []struct {
			Name       string `json:"name"`
			Deprecated bool   `json:"deprecated"`
		} `json:"commands"`
		Error  map[string]interface{}   `json:"error"`
		Global []map[string]interface{} `json:"global_options"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	if payload.Version == "" {
		t.Fatalf("schema missing version")
	}
	if payload.Name != "issuesherpa" {
		t.Fatalf("schema name = %s, want issuesherpa", payload.Name)
	}
	if len(payload.Commands) == 0 {
		t.Fatalf("schema missing commands")
	}
	if len(payload.Global) == 0 {
		t.Fatalf("schema missing global_options")
	}
	if _, ok := payload.Error["fields"]; !ok {
		t.Fatalf("schema missing error contract")
	}
	var foundDeprecatedJSON bool
	for _, command := range payload.Commands {
		if command.Name == "json" {
			foundDeprecatedJSON = command.Deprecated
			break
		}
	}
	if !foundDeprecatedJSON {
		t.Fatalf("json command must be deprecated in schema")
	}
}

func TestRunCLISchemaForCommand(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"search", "--schema"}, sampleTestIssues())
	if err != nil {
		t.Fatalf("runCLI search --schema returned error: %v", err)
	}
	var payload struct {
		Version     string   `json:"version"`
		Name        string   `json:"name"`
		Command     string   `json:"command"`
		Description string   `json:"description"`
		Usage       string   `json:"usage"`
		OutputModes []string `json:"output_modes"`
		Fields      []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"fields"`
		Response map[string]interface{} `json:"response"`
		Error    map[string]interface{} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid command schema JSON: %v", err)
	}
	if payload.Version == "" {
		t.Fatalf("command schema missing version")
	}
	if payload.Name != "issuesherpa" {
		t.Fatalf("schema name = %s, want issuesherpa", payload.Name)
	}
	if payload.Command != "search" {
		t.Fatalf("command schema = %s, want search", payload.Command)
	}
	if payload.Description == "" {
		t.Fatalf("command schema missing description")
	}
	if payload.Usage == "" {
		t.Fatalf("command schema missing usage")
	}
	if len(payload.OutputModes) == 0 {
		t.Fatalf("command schema missing output_modes")
	}
	if len(payload.Fields) == 0 {
		t.Fatalf("command schema missing fields")
	}
	for _, field := range payload.Fields {
		if field.Name == "" {
			t.Fatalf("command schema field missing name")
		}
		if field.Type == "" {
			t.Fatalf("command field %q missing type", field.Name)
		}
	}
	if got, ok := payload.Response["type"]; !ok || got != "object" {
		t.Fatalf("command response.type = %v, want object", got)
	}
	if _, ok := payload.Response["fields"]; !ok {
		t.Fatalf("command schema missing response contract")
	}
	if _, ok := payload.Error["fields"]; !ok {
		t.Fatalf("command schema missing error contract")
	}
}

func TestRunCLISearchRequiresQuery(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"search"}, sampleTestIssues())
	if err == nil {
		t.Fatalf("expected search without query to fail")
	}
	assertCLIError(t, out, "SEARCH_QUERY_REQUIRED", "query", cliExitCodeUsage)
}

func TestRunCLIShowResolvesByID(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"show", issues[0].ShortID}, issues)
	if err != nil {
		t.Fatalf("runCLI show returned error: %v", err)
	}
	var payload struct {
		Command string                   `json:"command"`
		Count   int                      `json:"count"`
		Total   int                      `json:"total"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid show JSON: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(payload.Data))
	}
	if payload.Command != "show" {
		t.Fatalf("command = %s, want show", payload.Command)
	}
	if payload.Count != 1 {
		t.Fatalf("count = %d, want 1", payload.Count)
	}
	if payload.Total != 1 {
		t.Fatalf("total = %d, want 1", payload.Total)
	}
	if payload.Data[0]["short_id"] != issues[0].ShortID {
		t.Fatalf("show short_id = %v, want %v", payload.Data[0]["short_id"], issues[0].ShortID)
	}
}

func TestRunCLIShowOutputNDJSON(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"show", "--output", "ndjson", issues[0].ShortID}, issues)
	if err != nil {
		t.Fatalf("runCLI show ndjson returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("ndjson lines = %d, want 1", len(lines))
	}
	var payload struct {
		Command string                 `json:"command"`
		Count   int                    `json:"count"`
		Total   int                    `json:"total"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("invalid show ndjson: %v", err)
	}
	if payload.Command != "show" {
		t.Fatalf("command = %s, want show", payload.Command)
	}
	if payload.Count != 1 {
		t.Fatalf("count = %d, want 1", payload.Count)
	}
	if payload.Total != 1 {
		t.Fatalf("total = %d, want 1", payload.Total)
	}
	if payload.Data["short_id"] != issues[0].ShortID {
		t.Fatalf("short_id = %v, want %v", payload.Data["short_id"], issues[0].ShortID)
	}
}

func TestRunCLILeaderboardOutputNDJSON(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"leaderboard", "--output", "ndjson"}, issues)
	if err != nil {
		t.Fatalf("runCLI leaderboard ndjson returned error: %v", err)
	}
	expectedRows := 0
	seen := make(map[string]struct{})
	for _, issue := range issues {
		if _, ok := seen[issue.Reporter]; !ok {
			seen[issue.Reporter] = struct{}{}
			expectedRows++
		}
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != expectedRows {
		t.Fatalf("leaderboard ndjson lines = %d, want %d", len(lines), expectedRows)
	}
	for _, line := range lines {
		var payload struct {
			Command string                 `json:"command"`
			Count   int                    `json:"count"`
			Total   int                    `json:"total"`
			Data    map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("invalid leaderboard ndjson: %v", err)
		}
		if payload.Command != "leaderboard" {
			t.Fatalf("command = %s, want leaderboard", payload.Command)
		}
		if payload.Count != 1 {
			t.Fatalf("count = %d, want 1", payload.Count)
		}
		if payload.Total != expectedRows {
			t.Fatalf("total = %d, want %d", payload.Total, expectedRows)
		}
		if _, ok := payload.Data["reporter"]; !ok {
			t.Fatalf("missing reporter in ndjson line: %q", line)
		}
	}
}

func TestRunCLIShowFieldSelection(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"show", "--fields", "id,short_id,source", issues[0].ShortID}, issues)
	if err != nil {
		t.Fatalf("runCLI show with fields returned error: %v", err)
	}
	var payload struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid show JSON: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(payload.Data))
	}
	if _, ok := payload.Data[0]["id"]; !ok {
		t.Fatalf("expected id in selected fields")
	}
	if _, ok := payload.Data[0]["short_id"]; !ok {
		t.Fatalf("expected short_id in selected fields")
	}
	if _, ok := payload.Data[0]["source"]; !ok {
		t.Fatalf("expected source in selected fields")
	}
	if _, ok := payload.Data[0]["title"]; ok {
		t.Fatalf("did not expect title in selected fields")
	}
}

func TestRunCLILimitPagination(t *testing.T) {
	issues := sampleTestIssues()
	out, _, err := runCLICaptureOutput(t, []string{"list", "--limit", "1", "--offset", "1"}, issues)
	if err != nil {
		t.Fatalf("runCLI list --limit returned error: %v", err)
	}
	var payload struct {
		Count int `json:"count"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.Count != 1 {
		t.Fatalf("count = %d, want 1", payload.Count)
	}
	if payload.Total != len(issues) {
		t.Fatalf("total = %d, want %d", payload.Total, len(issues))
	}
}

func TestRunCLISchemaRejectsBadOutputMode(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"list", "--output", "xml"}, sampleTestIssues())
	if err == nil {
		t.Fatalf("expected invalid output mode to fail")
	}
	assertCLIError(t, out, "INVALID_ARGUMENT", "unknown output mode", cliExitCodeUsage)
}

func TestRunCLIUnknownCommand(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"does-not-exist"}, sampleTestIssues())
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	assertCLIError(t, out, "UNKNOWN_COMMAND", "unknown command", cliExitCodeUsage)
}

func TestRunCLIUnknownCommandNDJSON(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"--output", "ndjson", "does-not-exist"}, sampleTestIssues())
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	assertCLIError(t, out, "UNKNOWN_COMMAND", "unknown command", cliExitCodeUsage)
}

func TestRunCLIShowMissingIssue(t *testing.T) {
	out, _, err := runCLICaptureOutput(t, []string{"show", "missing:999"}, sampleTestIssues())
	if err == nil {
		t.Fatalf("expected missing issue to fail")
	}
	assertCLIError(t, out, "ISSUE_NOT_FOUND", "not found", cliExitCodeNotFound)
}

func TestRunCLIEnvOutputDefaults(t *testing.T) {
	t.Setenv("ISSUESHERPA_OUTPUT", "text")
	out, _, err := runCLICaptureOutput(t, []string{"list"}, sampleTestIssues())
	if err != nil {
		t.Fatalf("runCLI list returned error: %v", err)
	}
	if !strings.Contains(out, "SOURCE") {
		t.Fatalf("expected env output mode text, got JSON or no output: %q", out)
	}
}

func TestRunCLIPrettyEnvHonorsOutputEvenWithNoColor(t *testing.T) {
	t.Setenv("ISSUESHERPA_PRETTY", "1")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("ISSUESHERPA_OUTPUT", "json")
	out, _, err := runCLICaptureOutput(t, []string{"--schema"}, sampleTestIssues())
	if err != nil {
		t.Fatalf("runCLI --schema returned error: %v", err)
	}
	if strings.Count(out, "\n") <= 1 {
		t.Fatalf("expected pretty JSON output when ISSUESHERPA_PRETTY=1, got compact: %q", out)
	}
}

func TestRunCLIDescribeIsDeprecated(t *testing.T) {
	out, stderr, err := runCLICaptureOutput(t, []string{"--describe", "list"}, sampleTestIssues())
	if err != nil {
		t.Fatalf("runCLI --describe list returned error: %v", err)
	}
	if out == "" {
		t.Fatalf("expected schema output")
	}
	if !strings.Contains(stderr, "deprecated") {
		t.Fatalf("expected deprecation warning, got %q", stderr)
	}
}

func assertCLIError(t *testing.T, out string, expectedCode string, expectedMessage string, expectedExitCode int) {
	t.Helper()
	var payload struct {
		Error struct {
			Code     string `json:"code"`
			Message  string `json:"message"`
			ExitCode int    `json:"exit_code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid error JSON: %v", err)
	}
	if payload.Error.Code != expectedCode {
		t.Fatalf("error code = %s, want %s", payload.Error.Code, expectedCode)
	}
	if !strings.Contains(strings.ToLower(payload.Error.Message), strings.ToLower(expectedMessage)) {
		t.Fatalf("error message = %q, want to include %q", payload.Error.Message, expectedMessage)
	}
	if payload.Error.ExitCode != expectedExitCode {
		t.Fatalf("exit_code = %d, want %d", payload.Error.ExitCode, expectedExitCode)
	}
}

func runCLICaptureOutput(t *testing.T, args []string, issues []Issue) (string, string, error) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&outBuf, rOut)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&errBuf, rErr)
	}()

	errVal := runCLI(args, issues)

	_ = wOut.Close()
	_ = wErr.Close()
	wg.Wait()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return outBuf.String(), errBuf.String(), errVal
}

func sampleTestIssues() []Issue {
	return []Issue{
		{
			ID:         "github:100",
			ShortID:    "frontend#100",
			Title:      "Fix login bug in frontend",
			Status:     "open",
			Level:      "error",
			Project:    Project{ID: "p1", Name: "Frontend", Slug: "frontend"},
			Count:      "42",
			UserCount:  3,
			FirstSeen:  "2026-01-01T10:00:00Z",
			LastSeen:   "2026-01-02T11:00:00Z",
			Reporter:   "alice",
			AssignedTo: &AssignedTo{Name: "Alice", Email: "alice@example.com"},
			Source:     "github",
			URL:        "https://example/github.com/issues/100",
		},
		{
			ID:        "sentry:200",
			ShortID:   "backend#200",
			Title:     "Unhandled exception in API",
			Status:    "resolved",
			Level:     "error",
			Project:   Project{ID: "p2", Name: "Backend", Slug: "backend"},
			Count:     "9",
			UserCount: 1,
			FirstSeen: "2026-01-01T12:00:00Z",
			LastSeen:  "2026-01-01T13:00:00Z",
			Reporter:  "bob",
			Source:    "sentry",
			URL:       "https://example/sentry/backend/200",
		},
		{
			ID:        "gitlab:300",
			ShortID:   "platform#300",
			Title:     "Slow query in DB layer",
			Status:    "open",
			Level:     "warning",
			Project:   Project{ID: "p3", Name: "Platform", Slug: "platform"},
			Count:     "8",
			UserCount: 2,
			FirstSeen: "2026-01-03T09:00:00Z",
			LastSeen:  "2026-01-03T10:00:00Z",
			Reporter:  "carol",
			Source:    "gitlab",
			URL:       "https://example/gitlab.com/issues/300",
		},
	}
}
