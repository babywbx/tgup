package plan

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/wbx/tgup/internal/scan"
)

func TestBuildSortByNameAndMTime(t *testing.T) {
	t.Parallel()

	items := []scan.Item{
		{Path: "/tmp/c.mp4", SrcRoot: "/tmp", ParentDir: "event", MTimeNS: 30, Size: 200},
		{Path: "/tmp/a.jpg", SrcRoot: "/tmp", ParentDir: "event", MTimeNS: 20, Size: 300},
		{Path: "/tmp/b.jpg", SrcRoot: "/tmp", ParentDir: "event", MTimeNS: 10, Size: 100},
	}

	byName := Build(items, Options{Order: "name", AlbumMax: 10})
	if got := collectPaths(byName); !slices.Equal(got, []string{"/tmp/a.jpg", "/tmp/b.jpg", "/tmp/c.mp4"}) {
		t.Fatalf("unexpected name order: %#v", got)
	}

	byMTimeReverse := Build(items, Options{Order: "mtime", Reverse: true, AlbumMax: 10})
	if got := collectPaths(byMTimeReverse); !slices.Equal(got, []string{"/tmp/c.mp4", "/tmp/a.jpg", "/tmp/b.jpg"}) {
		t.Fatalf("unexpected mtime reverse order: %#v", got)
	}

	bySize := Build(items, Options{Order: "size", AlbumMax: 10})
	if got := collectPaths(bySize); !slices.Equal(got, []string{"/tmp/b.jpg", "/tmp/c.mp4", "/tmp/a.jpg"}) {
		t.Fatalf("unexpected size order: %#v", got)
	}
}

func TestBuildGroupAndSlice(t *testing.T) {
	t.Parallel()

	items := []scan.Item{
		{Path: "/root/holiday/a.jpg", SrcRoot: "/root", ParentDir: "holiday"},
		{Path: "/root/holiday/b.jpg", SrcRoot: "/root", ParentDir: "holiday"},
		{Path: "/root/holiday/c.jpg", SrcRoot: "/root", ParentDir: "holiday"},
	}

	plan := Build(items, Options{Order: "name", AlbumMax: 2})
	if len(plan.Albums) != 2 {
		t.Fatalf("expected 2 albums, got %d", len(plan.Albums))
	}
	if plan.Albums[0].Label != "holiday (1/2)" {
		t.Fatalf("unexpected first label: %q", plan.Albums[0].Label)
	}
	if plan.Albums[1].Label != "holiday (2/2)" {
		t.Fatalf("unexpected second label: %q", plan.Albums[1].Label)
	}
	if len(plan.Albums[0].Items) != 2 || len(plan.Albums[1].Items) != 1 {
		t.Fatalf("unexpected album split sizes: %d, %d", len(plan.Albums[0].Items), len(plan.Albums[1].Items))
	}
}

func TestBuildSameParentDifferentRoots(t *testing.T) {
	t.Parallel()

	items := []scan.Item{
		{Path: "/root1/event/a.jpg", SrcRoot: "/root1", ParentDir: "event"},
		{Path: "/root2/event/b.jpg", SrcRoot: "/root2", ParentDir: "event"},
	}

	plan := Build(items, Options{Order: "name", AlbumMax: 10})
	if len(plan.Albums) != 2 {
		t.Fatalf("expected two albums for different src roots, got %d", len(plan.Albums))
	}
}

func TestBuildRootLabelUsesSrcRootBase(t *testing.T) {
	t.Parallel()

	srcRoot := filepath.Clean("/tmp/source")
	items := []scan.Item{
		{Path: "/tmp/source/a.jpg", SrcRoot: srcRoot, ParentDir: ""},
	}

	plan := Build(items, Options{})
	if len(plan.Albums) != 1 {
		t.Fatalf("expected one album, got %d", len(plan.Albums))
	}
	if plan.Albums[0].Label != "source" {
		t.Fatalf("expected root label source, got %q", plan.Albums[0].Label)
	}
}

func collectPaths(plan Plan) []string {
	out := make([]string, 0)
	for _, album := range plan.Albums {
		for _, item := range album.Items {
			out = append(out, item.Path)
		}
	}
	return out
}
