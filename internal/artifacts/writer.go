package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LogSink writes structured runtime logs.
type LogSink interface {
	Write(level string, message string) error
	Close() error
}

// ReportWriter writes report outputs for a run.
type ReportWriter interface {
	WriteJSON(summary any) error
	WriteMarkdown(summary any) error
}

// FileReportWriter writes report outputs to files.
type FileReportWriter struct {
	JSONPath     string
	MarkdownPath string
}

// NewFileReportWriter creates a report writer rooted at dir.
func NewFileReportWriter(dir string) FileReportWriter {
	clean := filepath.Clean(dir)
	return FileReportWriter{
		JSONPath:     filepath.Join(clean, "report.json"),
		MarkdownPath: filepath.Join(clean, "report.md"),
	}
}

// WriteJSON writes the summary as pretty JSON.
func (w FileReportWriter) WriteJSON(summary any) error {
	if err := os.MkdirAll(filepath.Dir(w.JSONPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report json: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(w.JSONPath, raw, 0o644); err != nil {
		return fmt.Errorf("write report json: %w", err)
	}
	return nil
}

// WriteMarkdown writes the summary as markdown.
func (w FileReportWriter) WriteMarkdown(summary any) error {
	if err := os.MkdirAll(filepath.Dir(w.MarkdownPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	text, err := RenderMarkdown(summary)
	if err != nil {
		return err
	}
	if err := os.WriteFile(w.MarkdownPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write report markdown: %w", err)
	}
	return nil
}
