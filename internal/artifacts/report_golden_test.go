package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDryRunSummaryMarkdownGolden(t *testing.T) {
	t.Parallel()

	summary := DryRunSummary{
		Sources:  []string{"/src/a", "/src/b"},
		Items:    3,
		Albums:   2,
		Order:    "name",
		Reverse:  false,
		AlbumMax: 10,
		Preview: []AlbumPreview{
			{Label: "album-1", Count: 2, Items: []string{"/src/a/1.jpg", "/src/a/2.mp4"}},
			{Label: "album-2", Count: 1, Items: []string{"/src/b/3.jpg"}},
		},
	}

	got := summary.Markdown()
	want := mustReadGoldenReport(t, "reports/dryrun_report.md")
	if got != want {
		t.Fatalf("markdown golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestDryRunSummaryJSONGolden(t *testing.T) {
	t.Parallel()

	summary := DryRunSummary{
		Sources:  []string{"/src/a", "/src/b"},
		Items:    3,
		Albums:   2,
		Order:    "name",
		Reverse:  false,
		AlbumMax: 10,
		Preview: []AlbumPreview{
			{Label: "album-1", Count: 2, Items: []string{"/src/a/1.jpg", "/src/a/2.mp4"}},
			{Label: "album-2", Count: 1, Items: []string{"/src/b/3.jpg"}},
		},
	}

	dir := t.TempDir()
	writer := NewFileReportWriter(dir)
	if err := writer.WriteJSON(summary); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	raw, err := os.ReadFile(writer.JSONPath)
	if err != nil {
		t.Fatalf("read report json: %v", err)
	}

	want := mustReadGoldenReport(t, "reports/dryrun_report.json")
	if string(raw) != want {
		t.Fatalf("json golden mismatch\n--- got ---\n%s\n--- want ---\n%s", string(raw), want)
	}
}

func mustReadGoldenReport(t *testing.T, rel string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Join(cwd, "..", "..", "testdata", "golden", rel)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v", path, err)
	}
	return string(raw)
}
