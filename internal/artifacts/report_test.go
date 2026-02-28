package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReportWriterWritesJSONAndMarkdown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writer := NewFileReportWriter(dir)
	summary := DryRunSummary{
		Sources:  []string{"/tmp/a"},
		Items:    3,
		Albums:   2,
		Order:    "name",
		Reverse:  false,
		AlbumMax: 10,
		Preview: []AlbumPreview{
			{Label: "album-1", Count: 2, Items: []string{"/tmp/a/1.jpg", "/tmp/a/2.jpg"}},
		},
	}

	if err := writer.WriteJSON(summary); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if err := writer.WriteMarkdown(summary); err != nil {
		t.Fatalf("WriteMarkdown() error = %v", err)
	}

	jsonRaw, err := os.ReadFile(writer.JSONPath)
	if err != nil {
		t.Fatalf("read report json: %v", err)
	}
	mdRaw, err := os.ReadFile(writer.MarkdownPath)
	if err != nil {
		t.Fatalf("read report markdown: %v", err)
	}

	if !strings.Contains(string(jsonRaw), `"items": 3`) {
		t.Fatalf("unexpected json output: %s", string(jsonRaw))
	}
	if !strings.Contains(string(mdRaw), "# Dry Run Report") {
		t.Fatalf("unexpected markdown output: %s", string(mdRaw))
	}
}

func TestNewFileReportWriterPaths(t *testing.T) {
	t.Parallel()

	writer := NewFileReportWriter("/tmp/out")
	if writer.JSONPath != filepath.Clean("/tmp/out/report.json") {
		t.Fatalf("unexpected json path: %q", writer.JSONPath)
	}
	if writer.MarkdownPath != filepath.Clean("/tmp/out/report.md") {
		t.Fatalf("unexpected markdown path: %q", writer.MarkdownPath)
	}
}
