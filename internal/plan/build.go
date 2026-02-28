package plan

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wbx/tgup/internal/scan"
)

const defaultAlbumMax = 10

// Options controls how scan items are transformed into upload albums.
type Options struct {
	Order    string
	Reverse  bool
	AlbumMax int
}

type groupKey struct {
	srcRoot   string
	parentDir string
}

// Build groups and slices media items into a deterministic upload plan.
func Build(items []scan.Item, opts Options) Plan {
	if len(items) == 0 {
		return Plan{}
	}

	sorted := cloneItems(items)
	sortItems(sorted, opts.Order, opts.Reverse)

	maxPerAlbum := opts.AlbumMax
	if maxPerAlbum <= 0 {
		maxPerAlbum = defaultAlbumMax
	}

	grouped := make(map[groupKey][]scan.Item)
	order := make([]groupKey, 0)
	for _, item := range sorted {
		key := groupKey{
			srcRoot:   item.SrcRoot,
			parentDir: item.ParentDir,
		}
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], item)
	}

	albums := make([]Album, 0)
	for _, key := range order {
		groupItems := grouped[key]
		label := buildLabel(key)
		chunks := split(groupItems, maxPerAlbum)
		if len(chunks) == 1 {
			albums = append(albums, Album{
				Label: label,
				Items: chunks[0],
			})
			continue
		}
		for i, chunk := range chunks {
			albums = append(albums, Album{
				Label: fmt.Sprintf("%s (%d/%d)", label, i+1, len(chunks)),
				Items: chunk,
			})
		}
	}
	return Plan{Albums: albums}
}

func cloneItems(items []scan.Item) []scan.Item {
	out := make([]scan.Item, len(items))
	copy(out, items)
	return out
}

func sortItems(items []scan.Item, order string, reverse bool) {
	order = strings.ToLower(strings.TrimSpace(order))
	switch order {
	case "mtime":
		sort.Slice(items, func(i, j int) bool {
			if items[i].MTimeNS == items[j].MTimeNS {
				return items[i].Path < items[j].Path
			}
			return items[i].MTimeNS < items[j].MTimeNS
		})
	case "size":
		sort.Slice(items, func(i, j int) bool {
			if items[i].Size == items[j].Size {
				return items[i].Path < items[j].Path
			}
			return items[i].Size < items[j].Size
		})
	case "random":
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(items), func(i int, j int) {
			items[i], items[j] = items[j], items[i]
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			if items[i].Path == items[j].Path {
				return items[i].MTimeNS < items[j].MTimeNS
			}
			return items[i].Path < items[j].Path
		})
	}

	if !reverse {
		return
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func split(items []scan.Item, size int) [][]scan.Item {
	if len(items) == 0 {
		return nil
	}

	chunks := make([][]scan.Item, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunk := make([]scan.Item, end-start)
		copy(chunk, items[start:end])
		chunks = append(chunks, chunk)
	}
	return chunks
}

func buildLabel(key groupKey) string {
	if key.parentDir != "" {
		return filepath.ToSlash(key.parentDir)
	}
	base := filepath.Base(key.srcRoot)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return filepath.ToSlash(key.srcRoot)
	}
	return filepath.ToSlash(base)
}
