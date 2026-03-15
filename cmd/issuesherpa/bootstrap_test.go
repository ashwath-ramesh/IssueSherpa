package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := version, commit, buildDate
	version, commit, buildDate = "v1.2.3", "abc1234", "2026-03-15T12:00:00Z"
	defer func() {
		version, commit, buildDate = oldVersion, oldCommit, oldBuildDate
	}()

	got := formatVersion()
	want := "issuesherpa v1.2.3 (commit abc1234, built 2026-03-15T12:00:00Z)"
	if got != want {
		t.Fatalf("formatVersion = %q, want %q", got, want)
	}
}

func TestFormatVersionAddsVPrefixForSemver(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := version, commit, buildDate
	version, commit, buildDate = "1.2.3", "abc1234", "2026-03-15T12:00:00Z"
	defer func() {
		version, commit, buildDate = oldVersion, oldCommit, oldBuildDate
	}()

	got := formatVersion()
	want := "issuesherpa v1.2.3 (commit abc1234, built 2026-03-15T12:00:00Z)"
	if got != want {
		t.Fatalf("formatVersion = %q, want %q", got, want)
	}
}

func TestHandleBootstrapCommandVersion(t *testing.T) {
	stdout := captureStdout(t, func() {
		handled, exitCode := handleBootstrapCommand([]string{"--version"})
		if !handled {
			t.Fatal("expected version command to be handled")
		}
		if exitCode != 0 {
			t.Fatalf("exitCode = %d, want 0", exitCode)
		}
	})
	if !strings.Contains(stdout, "issuesherpa ") {
		t.Fatalf("expected version output, got %q", stdout)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	_ = r.Close()
	return string(data)
}
