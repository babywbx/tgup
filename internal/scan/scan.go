package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Options controls discovery behavior.
type Options struct {
	Src            []string
	Recursive      bool
	FollowSymlinks bool
	IncludeExt     []string
	ExcludeExt     []string
}

var (
	defaultImageExt = map[string]struct{}{
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".webp": {},
		".heic": {},
	}
	defaultVideoExt = map[string]struct{}{
		".mp4":  {},
		".mov":  {},
		".mkv":  {},
		".webm": {},
	}
)

// Discover scans configured source paths and returns normalized media items.
func Discover(opts Options) ([]Item, error) {
	if len(opts.Src) == 0 {
		return nil, nil
	}

	include := buildExtSet(opts.IncludeExt)
	exclude := buildExtSet(opts.ExcludeExt)
	seenFiles := make(map[string]struct{})
	seenDirs := make(map[string]struct{})
	items := make([]Item, 0)

	for _, source := range opts.Src {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}

		absSource, err := filepath.Abs(source)
		if err != nil {
			return nil, fmt.Errorf("resolve source path %q: %w", source, err)
		}

		srcRoot, err := sourceRoot(absSource, opts.FollowSymlinks)
		if err != nil {
			return nil, err
		}

		if err := scanPath(scanContext{
			path:     absSource,
			srcRoot:  srcRoot,
			opts:     opts,
			include:  include,
			exclude:  exclude,
			seenDirs: seenDirs,
			seenFile: seenFiles,
			out:      &items,
		}); err != nil {
			return nil, err
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})
	return items, nil
}

type scanContext struct {
	path     string
	srcRoot  string
	opts     Options
	include  map[string]struct{}
	exclude  map[string]struct{}
	seenDirs map[string]struct{}
	seenFile map[string]struct{}
	out      *[]Item
}

func scanPath(ctx scanContext) error {
	info, err := os.Lstat(ctx.path)
	if err != nil {
		return fmt.Errorf("lstat %q: %w", ctx.path, err)
	}

	if isSymlink(info) {
		if !ctx.opts.FollowSymlinks {
			return nil
		}
		info, err = os.Stat(ctx.path)
		if err != nil {
			return fmt.Errorf("stat symlink %q: %w", ctx.path, err)
		}
	}

	if info.IsDir() {
		return scanDir(ctx)
	}
	return maybeAppendItem(ctx, info)
}

func scanDir(ctx scanContext) error {
	key := ctx.path
	if ctx.opts.FollowSymlinks {
		if resolved, err := filepath.EvalSymlinks(ctx.path); err == nil {
			key = resolved
		}
	}
	if _, seen := ctx.seenDirs[key]; seen {
		return nil
	}
	ctx.seenDirs[key] = struct{}{}

	entries, err := os.ReadDir(ctx.path)
	if err != nil {
		return fmt.Errorf("read dir %q: %w", ctx.path, err)
	}

	for _, entry := range entries {
		child := filepath.Join(ctx.path, entry.Name())
		info, err := os.Lstat(child)
		if err != nil {
			return fmt.Errorf("lstat %q: %w", child, err)
		}

		isLink := isSymlink(info)
		if isLink && !ctx.opts.FollowSymlinks {
			continue
		}
		if isLink {
			info, err = os.Stat(child)
			if err != nil {
				// Broken symlinks are ignored during discovery.
				continue
			}
		}

		if info.IsDir() {
			if ctx.opts.Recursive {
				next := ctx
				next.path = child
				if err := scanDir(next); err != nil {
					return err
				}
			}
			continue
		}
		if err := maybeAppendItem(scanContext{
			path:     child,
			srcRoot:  ctx.srcRoot,
			opts:     ctx.opts,
			include:  ctx.include,
			exclude:  ctx.exclude,
			seenDirs: ctx.seenDirs,
			seenFile: ctx.seenFile,
			out:      ctx.out,
		}, info); err != nil {
			return err
		}
	}
	return nil
}

func maybeAppendItem(ctx scanContext, info fs.FileInfo) error {
	if !info.Mode().IsRegular() {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(ctx.path))
	if !extAllowed(ext, ctx.include, ctx.exclude) {
		return nil
	}

	kind, ok := detectKind(ext)
	if !ok {
		return nil
	}

	absPath, err := filepath.Abs(ctx.path)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", ctx.path, err)
	}
	// Always resolve symlinks for dedup key.
	dedupKey := absPath
	if real, evalErr := filepath.EvalSymlinks(absPath); evalErr == nil {
		dedupKey = filepath.Clean(real)
	}

	if _, seen := ctx.seenFile[dedupKey]; seen {
		return nil
	}
	ctx.seenFile[dedupKey] = struct{}{}

	// Item.Path uses the resolved path when following symlinks.
	itemPath := absPath
	if ctx.opts.FollowSymlinks {
		itemPath = dedupKey
	}

	parent := parentDir(dedupKey, ctx.srcRoot)
	*ctx.out = append(*ctx.out, Item{
		Path:      itemPath,
		SrcRoot:   ctx.srcRoot,
		ParentDir: parent,
		Size:      info.Size(),
		MTimeNS:   info.ModTime().UnixNano(),
		Kind:      kind,
	})
	return nil
}

func sourceRoot(source string, followSymlinks bool) (string, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return "", fmt.Errorf("lstat %q: %w", source, err)
	}

	if isSymlink(info) {
		target, statErr := os.Stat(source)
		if statErr != nil {
			if followSymlinks {
				return "", fmt.Errorf("stat symlink %q: %w", source, statErr)
			}
			// Broken symlink with follow disabled; treat as file.
		} else {
			info = target
		}
	}

	root := source
	if !info.IsDir() {
		root = filepath.Dir(source)
	}
	root = filepath.Clean(root)

	if followSymlinks {
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = filepath.Clean(resolved)
		}
	}
	return root, nil
}

func parentDir(path string, root string) string {
	relative, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil || strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(filepath.Base(filepath.Dir(path)))
	}
	if relative == "." {
		return ""
	}
	return filepath.ToSlash(relative)
}

func extAllowed(ext string, include map[string]struct{}, exclude map[string]struct{}) bool {
	if _, blocked := exclude[ext]; blocked {
		return false
	}
	if len(include) == 0 {
		return true
	}
	_, allowed := include[ext]
	return allowed
}

func detectKind(ext string) (Kind, bool) {
	if _, ok := defaultImageExt[ext]; ok {
		return KindImage, true
	}
	if _, ok := defaultVideoExt[ext]; ok {
		return KindVideo, true
	}
	return "", false
}

func buildExtSet(exts []string) map[string]struct{} {
	set := make(map[string]struct{}, len(exts))
	for _, ext := range exts {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		set[ext] = struct{}{}
	}
	return set
}

func isSymlink(info fs.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
