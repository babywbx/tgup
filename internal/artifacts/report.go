package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AlbumPreview captures a short per-album preview.
type AlbumPreview struct {
	Label string   `json:"label"`
	Count int      `json:"count"`
	Items []string `json:"items"`
}

// DryRunSummary is a serializable summary for dry-run reporting.
type DryRunSummary struct {
	Sources  []string       `json:"sources"`
	Items    int            `json:"items"`
	Albums   int            `json:"albums"`
	Order    string         `json:"order"`
	Reverse  bool           `json:"reverse"`
	AlbumMax int            `json:"album_max"`
	Preview  []AlbumPreview `json:"preview"`
}

// RenderMarkdown converts known report summaries to markdown.
func RenderMarkdown(summary any) (string, error) {
	switch s := summary.(type) {
	case DryRunSummary:
		return s.Markdown(), nil
	case *DryRunSummary:
		if s == nil {
			return "", fmt.Errorf("render markdown: nil summary")
		}
		return s.Markdown(), nil
	default:
		raw, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return "", fmt.Errorf("render markdown: marshal fallback json: %w", err)
		}
		return "```json\n" + string(raw) + "\n```\n", nil
	}
}

// Markdown renders a human-readable markdown summary.
func (s DryRunSummary) Markdown() string {
	var b strings.Builder
	b.WriteString("# Dry Run Report\n\n")
	b.WriteString(fmt.Sprintf("- sources: %d\n", len(s.Sources)))
	b.WriteString(fmt.Sprintf("- items: %d\n", s.Items))
	b.WriteString(fmt.Sprintf("- albums: %d\n", s.Albums))
	b.WriteString(fmt.Sprintf("- order: %s\n", s.Order))
	b.WriteString(fmt.Sprintf("- reverse: %t\n", s.Reverse))
	b.WriteString(fmt.Sprintf("- album_max: %d\n", s.AlbumMax))
	b.WriteString("\n## Preview\n")
	if len(s.Preview) == 0 {
		b.WriteString("\n(no albums)\n")
		return b.String()
	}
	b.WriteString("\n")
	for i, album := range s.Preview {
		b.WriteString(fmt.Sprintf("%d. %s (%d)\n", i+1, album.Label, album.Count))
		for _, item := range album.Items {
			b.WriteString(fmt.Sprintf("   - %s\n", item))
		}
	}
	return b.String()
}
