package scan

import "strings"

// Filter applies include/exclude extension rules.
type Filter struct {
	include map[string]struct{}
	exclude map[string]struct{}
}

// NewFilter creates extension filter from include/exclude lists.
func NewFilter(includeExt []string, excludeExt []string) Filter {
	return Filter{
		include: buildExtSet(includeExt),
		exclude: buildExtSet(excludeExt),
	}
}

// Allowed checks whether extension passes filter rules.
func (f Filter) Allowed(ext string) bool {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return false
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return extAllowed(ext, f.include, f.exclude)
}
