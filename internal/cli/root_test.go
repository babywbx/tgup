package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRunSuccess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "media"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "media", "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	cfgPath := filepath.Join(root, "tgup.toml")
	cfgBody := `
[telegram]
api_id = 12345
api_hash = "abc123"

[scan]
src = ["media"]
recursive = true

[plan]
order = "name"
album_max = 10
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dry-run", "--config", cfgPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "dry-run summary") || !strings.Contains(out, "items: 1") {
		t.Fatalf("unexpected dry-run output:\n%s", out)
	}
}

func TestRunDryRunWritesReports(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "media"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "media", "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	cfgPath := filepath.Join(root, "tgup.toml")
	cfgBody := `
[telegram]
api_id = 12345
api_hash = "abc123"

[scan]
src = ["media"]
recursive = true

[plan]
order = "name"
album_max = 10
`
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	reportDir := filepath.Join(root, "reports")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dry-run", "--config", cfgPath, "--report-dir", reportDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(reportDir, "report.json")); err != nil {
		t.Fatalf("expected report.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "report.md")); err != nil {
		t.Fatalf("expected report.md to exist: %v", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected unknown command error, got: %s", stderr.String())
	}
}

func TestRunKnownButUnimplementedCommands(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{"login"},
		{"run"},
		{"mcp", "serve"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(strings.Join(tc, "_"), func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("expected exit code 1, got %d", code)
			}
			if !strings.Contains(stderr.String(), "not implemented yet") {
				t.Fatalf("expected not implemented output, got: %s", stderr.String())
			}
		})
	}
}

func TestRunMCPSchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"mcp", "schema"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, `"version"`) || !strings.Contains(out, `"tools"`) {
		t.Fatalf("expected schema json output, got: %s", out)
	}
}
