package plan

import "github.com/wbx/tgup/internal/scan"

// SortItems returns a sorted copy of items using plan options.
func SortItems(items []scan.Item, order string, reverse bool) []scan.Item {
	sorted := cloneItems(items)
	sortItems(sorted, order, reverse)
	return sorted
}
