package progress

import (
	"fmt"
	"io"
)

// TerminalRenderer writes line-based progress updates.
type TerminalRenderer struct {
	out io.Writer
}

// NewTerminalRenderer builds a terminal renderer.
func NewTerminalRenderer(out io.Writer) *TerminalRenderer {
	return &TerminalRenderer{out: out}
}

// Write renders one snapshot to output.
func (r *TerminalRenderer) Write(s Snapshot) {
	if r == nil || r.out == nil {
		return
	}
	_, _ = fmt.Fprintln(r.out, RenderLine(s))
}
