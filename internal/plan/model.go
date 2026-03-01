package plan

import "github.com/babywbx/tgup/internal/scan"

// Album is the unit of upload scheduling.
type Album struct {
	Label string
	Items []scan.Item
}

// Plan is the full upload plan built from scanned items.
type Plan struct {
	Albums []Album
}
