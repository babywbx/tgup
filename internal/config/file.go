package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/babywbx/tgup/internal/files"
)

// LoadedConfig keeps parsed file overrides and source metadata.
type LoadedConfig struct {
	Overlay Overlay
	Path    string
	Dir     string
}

// LoadFile parses a TOML config file and resolves relative paths against file dir.
func LoadFile(path string) (LoadedConfig, error) {
	if path == "" {
		return LoadedConfig{}, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return LoadedConfig{}, fmt.Errorf("resolve config path: %w", err)
	}

	raw, err := os.ReadFile(absPath)
	if err != nil {
		return LoadedConfig{}, fmt.Errorf("read config file: %w", err)
	}

	var ov Overlay
	if err := decodeTOMLOverlay(raw, &ov); err != nil {
		return LoadedConfig{}, err
	}

	dir := filepath.Dir(absPath)
	resolveRelativePaths(&ov, dir)

	return LoadedConfig{
		Overlay: ov,
		Path:    absPath,
		Dir:     dir,
	}, nil
}

func decodeTOMLOverlay(raw []byte, target *Overlay) error {
	md, err := toml.Decode(string(raw), target)
	if err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	undecoded := md.Undecoded()
	if len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		sort.Strings(keys)
		return fmt.Errorf("parse config file: unknown fields: %s", strings.Join(keys, ", "))
	}
	return nil
}

func resolveRelativePaths(ov *Overlay, baseDir string) {
	resolvePath(&ov.Telegram.SessionPath, baseDir)
	resolvePath(&ov.Telegram.SessionPathLegacy, baseDir)
	resolvePath(&ov.Paths.StatePath, baseDir)
	resolvePath(&ov.Paths.StatePathLegacy, baseDir)
	resolvePath(&ov.Paths.ArtifactsDir, baseDir)
	resolvePathList(&ov.Scan.Src, baseDir)
	resolvePathMaybeValue(&ov.Upload.VideoThumbnail, baseDir, "auto", "off")
	resolvePath(&ov.Upload.StatePathCompat, baseDir)
	resolvePathList(&ov.MCP.AllowRoots, baseDir)
	resolvePath(&ov.MCP.ControlDB, baseDir)
}

func resolvePath(v **string, baseDir string) {
	if *v == nil {
		return
	}
	resolved := resolveOnePath(**v, baseDir)
	*v = &resolved
}

func resolvePathMaybeValue(v **string, baseDir string, skipValues ...string) {
	if *v == nil {
		return
	}
	val := **v
	for _, skip := range skipValues {
		if val == skip {
			return
		}
	}
	resolved := resolveOnePath(val, baseDir)
	*v = &resolved
}

func resolvePathList(v **[]string, baseDir string) {
	if *v == nil {
		return
	}
	src := **v
	out := make([]string, 0, len(src))
	for _, path := range src {
		out = append(out, resolveOnePath(path, baseDir))
	}
	*v = &out
}

func resolveOnePath(path string, baseDir string) string {
	return files.ResolveAgainst(path, baseDir)
}
