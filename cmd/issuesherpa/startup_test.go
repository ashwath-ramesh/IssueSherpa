package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestHandlePreRuntimeCommandTopLevelHelp(t *testing.T) {
	out, stderr, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"-h"})
	if !handled {
		t.Fatal("expected help to be handled before runtime")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected help output, got %q", out)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
}

func TestHandlePreRuntimeCommandSubcommandHelp(t *testing.T) {
	out, stderr, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"list", "-h"})
	if !handled {
		t.Fatal("expected subcommand help to be handled before runtime")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "Command: list") {
		t.Fatalf("expected command help, got %q", out)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
}

func TestHandlePreRuntimeCommandSchema(t *testing.T) {
	out, stderr, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"--schema"})
	if !handled {
		t.Fatal("expected schema to be handled before runtime")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "\"name\":\"issuesherpa\"") {
		t.Fatalf("expected schema output, got %q", out)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
}

func TestHandlePreRuntimeCommandSubcommandSchema(t *testing.T) {
	out, stderr, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"search", "--schema"})
	if !handled {
		t.Fatal("expected subcommand schema to be handled before runtime")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "\"command\":\"search\"") {
		t.Fatalf("expected command schema output, got %q", out)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
}

func TestHandlePreRuntimeCommandUnknownCommand(t *testing.T) {
	out, _, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"does-not-exist"})
	if !handled {
		t.Fatal("expected unknown command to fail before runtime")
	}
	if exitCode != cliExitCodeUsage {
		t.Fatalf("exitCode = %d, want %d", exitCode, cliExitCodeUsage)
	}
	if !strings.Contains(out, "\"code\":\"UNKNOWN_COMMAND\"") {
		t.Fatalf("expected unknown command error, got %q", out)
	}
}

func TestHandlePreRuntimeCommandUsageError(t *testing.T) {
	out, _, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"list", "--project"})
	if !handled {
		t.Fatal("expected usage error to fail before runtime")
	}
	if exitCode != cliExitCodeUsage {
		t.Fatalf("exitCode = %d, want %d", exitCode, cliExitCodeUsage)
	}
	if !strings.Contains(out, "\"code\": \"INVALID_ARGUMENT\"") {
		t.Fatalf("expected invalid argument error, got %q", out)
	}
}

func TestHandlePreRuntimeCommandValidRuntimeCommand(t *testing.T) {
	out, stderr, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"list"})
	if handled {
		t.Fatal("expected valid data command to continue to runtime")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if out != "" || stderr != "" {
		t.Fatalf("expected no output, got stdout=%q stderr=%q", out, stderr)
	}
}

func TestHandlePreRuntimeCommandOfflineRuntimeCommand(t *testing.T) {
	out, stderr, handled, exitCode := runPreRuntimeCaptureOutput(t, []string{"--offline", "list"})
	if handled {
		t.Fatal("expected valid offline data command to continue to runtime")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if out != "" || stderr != "" {
		t.Fatalf("expected no output, got stdout=%q stderr=%q", out, stderr)
	}
}

func runPreRuntimeCaptureOutput(t *testing.T, args []string) (string, string, bool, int) {
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

	handled, exitCode := handlePreRuntimeCommand(args)

	_ = wOut.Close()
	_ = wErr.Close()
	wg.Wait()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return outBuf.String(), errBuf.String(), handled, exitCode
}
