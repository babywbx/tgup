package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadFileResolvesRelativePaths(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	cfgPath := filepath.Join(baseDir, "tgup.toml")
	content := `
[telegram]
session = "sessions/main.session"

[paths]
state = "state/main.sqlite"
artifacts_dir = "artifacts"

[scan]
src = ["media", "/tmp/already-abs"]

[upload]
video_thumbnail = "thumbs/cover.jpg"

[mcp]
allow_roots = ["./media", "./data"]
control_db = "./data/mcp.sqlite"
`
	if err := osWriteFile(cfgPath, content); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	loaded, err := LoadFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	assertStringPtr(t, loaded.Overlay.Telegram.SessionPath, filepath.Join(baseDir, "sessions/main.session"))
	assertStringPtr(t, loaded.Overlay.Paths.StatePath, filepath.Join(baseDir, "state/main.sqlite"))
	assertStringPtr(t, loaded.Overlay.Paths.ArtifactsDir, filepath.Join(baseDir, "artifacts"))
	assertStringPtr(t, loaded.Overlay.Upload.VideoThumbnail, filepath.Join(baseDir, "thumbs/cover.jpg"))
	assertStringPtr(t, loaded.Overlay.MCP.ControlDB, filepath.Join(baseDir, "data/mcp.sqlite"))

	src := loaded.Overlay.Scan.Src
	if src == nil || len(*src) != 2 {
		t.Fatalf("expected 2 source paths, got %#v", src)
	}
	if got := (*src)[0]; got != filepath.Join(baseDir, "media") {
		t.Fatalf("expected relative src to resolve, got %q", got)
	}
	if got := (*src)[1]; got != "/tmp/already-abs" {
		t.Fatalf("expected absolute src to stay unchanged, got %q", got)
	}

	roots := loaded.Overlay.MCP.AllowRoots
	if roots == nil || len(*roots) != 2 {
		t.Fatalf("expected allow roots to load, got %#v", roots)
	}
	if got := (*roots)[0]; got != filepath.Join(baseDir, "media") {
		t.Fatalf("expected allow_root to resolve against config dir, got %q", got)
	}
}

func TestLoadFileRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	cfgPath := filepath.Join(baseDir, "tgup.toml")
	if err := osWriteFile(cfgPath, "[scan]\nunknown = true\n"); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	_, err := LoadFile(cfgPath)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown fields") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveWithOverlaysPrecedence(t *testing.T) {
	t.Parallel()

	defaults := Default()
	defaults.Telegram.APIID = 12345
	defaults.Telegram.APIHash = "abc123"
	defaults.Scan.Src = []string{"/default/src"}
	defaults.Paths.StatePath = "/default/state.sqlite"
	defaults.Plan.Order = "name"
	defaults.Upload.Duplicate = "skip"

	file := LoadedConfig{
		Overlay: Overlay{
			Paths: PathsOverlay{
				StatePath: strPtr("/file/state.sqlite"),
			},
			Plan: PlanOverlay{
				Order: strPtr("mtime"),
			},
			Upload: UploadOverlay{
				Duplicate: strPtr("ask"),
			},
		},
	}
	env := Overlay{
		Paths: PathsOverlay{
			StatePath: strPtr("/env/state.sqlite"),
		},
		Upload: UploadOverlay{
			Duplicate: strPtr("upload"),
		},
	}
	cli := Overlay{
		Upload: UploadOverlay{
			Duplicate: strPtr("skip"),
		},
	}

	resolved, err := ResolveWithOverlays(defaults, file, env, cli)
	if err != nil {
		t.Fatalf("ResolveWithOverlays() error = %v", err)
	}

	if resolved.Paths.StatePath != "/env/state.sqlite" {
		t.Fatalf("expected env to override file/default for state path, got %q", resolved.Paths.StatePath)
	}
	if resolved.Plan.Order != "mtime" {
		t.Fatalf("expected file to override default for plan.order, got %q", resolved.Plan.Order)
	}
	if resolved.Upload.Duplicate != "skip" {
		t.Fatalf("expected cli to override env/file/default for upload.duplicate, got %q", resolved.Upload.Duplicate)
	}
}

func TestLoadEnvParsesValuesAndCompatKeys(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"TGUP_SRC":               "a, b, c",
		"TGUP_RECURSIVE":         "no",
		"TGUP_INCLUDE_EXT":       "jpg,mp4",
		"TGUP_ALBUM_MAX":         "42",
		"TGUP_DUPLICATE":         "ask",
		"TGUP_MCP_ENABLED":       "on",
		"TGUP_MCP_PORT":          "9000",
		"TGUP_CONCURRENCY_ALBUM": "5",
		"TGUP_SESSION":           "/tmp/session.session",
		"TGUP_STATE":             "/tmp/state.sqlite",
	}
	ov, err := loadEnv(func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("loadEnv() error = %v", err)
	}

	if ov.Scan.Src == nil || len(*ov.Scan.Src) != 3 {
		t.Fatalf("expected src list from env, got %#v", ov.Scan.Src)
	}
	if ov.Scan.Recursive == nil || *ov.Scan.Recursive {
		t.Fatalf("expected recursive=false from env, got %#v", ov.Scan.Recursive)
	}
	if ov.Plan.AlbumMax == nil || *ov.Plan.AlbumMax != 42 {
		t.Fatalf("expected album max 42, got %#v", ov.Plan.AlbumMax)
	}
	if ov.MCP.Port == nil || *ov.MCP.Port != 9000 {
		t.Fatalf("expected mcp port 9000, got %#v", ov.MCP.Port)
	}
	if ov.Telegram.SessionPath == nil || *ov.Telegram.SessionPath != "/tmp/session.session" {
		t.Fatalf("expected session path from TGUP_SESSION, got %#v", ov.Telegram.SessionPath)
	}
	if ov.Paths.StatePath == nil || *ov.Paths.StatePath != "/tmp/state.sqlite" {
		t.Fatalf("expected state path from TGUP_STATE, got %#v", ov.Paths.StatePath)
	}
}

func TestLoadEnvRejectsConflictingCompatKeys(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"TGUP_SESSION":      "/tmp/a.session",
		"TGUP_SESSION_PATH": "/tmp/b.session",
	}
	_, err := loadEnv(func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	})
	if err == nil {
		t.Fatal("expected conflicting key error")
	}
	if !strings.Contains(err.Error(), "conflicting env values") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveUsesDefaultConfigChain(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	cwd := t.TempDir()

	globalCfg := filepath.Join(home, ".config", "tgup", "config.toml")
	projectCfg := filepath.Join(cwd, "tgup.toml")
	if err := osWriteFile(globalCfg, `
[telegram]
api_id = 12345
api_hash = "abc123"

[scan]
src = ["/global/src"]

[plan]
order = "name"

[upload]
duplicate = "upload"
`); err != nil {
		t.Fatalf("write global config: %v", err)
	}
	if err := osWriteFile(projectCfg, `
[scan]
src = ["./project-media"]

[plan]
order = "mtime"
`); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := resolve(
		"",
		Overlay{},
		func(string) (string, bool) { return "", false },
		func() (string, error) { return home, nil },
		func() (string, error) { return cwd, nil },
	)
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}

	wantSrc := []string{filepath.Join(cwd, "project-media")}
	if !slices.Equal(cfg.Scan.Src, wantSrc) {
		t.Fatalf("expected project src to override global, got %#v", cfg.Scan.Src)
	}
	if cfg.Plan.Order != "mtime" {
		t.Fatalf("expected project plan.order override, got %q", cfg.Plan.Order)
	}
	if cfg.Upload.Duplicate != "upload" {
		t.Fatalf("expected global duplicate fallback, got %q", cfg.Upload.Duplicate)
	}
}

func TestResolveExplicitConfigSkipsDefaultChain(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	cwd := t.TempDir()
	explicit := filepath.Join(cwd, "explicit.toml")
	if err := osWriteFile(explicit, `
[telegram]
api_id = 12345
api_hash = "abc123"

[scan]
src = ["./from-explicit"]
`); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	cfg, err := resolve(
		explicit,
		Overlay{},
		func(string) (string, bool) { return "", false },
		func() (string, error) { return home, nil },
		func() (string, error) { return cwd, nil },
	)
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}

	wantSrc := []string{filepath.Join(cwd, "from-explicit")}
	if !slices.Equal(cfg.Scan.Src, wantSrc) {
		t.Fatalf("expected explicit config src only, got %#v", cfg.Scan.Src)
	}
}

func TestValidateRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Plan.Order = "unknown"
	cfg.Plan.AlbumMax = 0
	cfg.Upload.ConcurrencyAlbum = 0
	cfg.Upload.ParseMode = "markdown"
	cfg.Upload.Duplicate = "invalid"
	cfg.Upload.VideoThumbnail = ""
	cfg.MCP.Enabled = true
	cfg.MCP.Port = 70000

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected Validate() to fail")
	}

	msg := err.Error()
	for _, needle := range []string{
		"telegram.api_id",
		"telegram.api_hash",
		"plan.order",
		"plan.album_max",
		"upload.concurrency_album",
		"upload.parse_mode",
		"upload.duplicate",
		"upload.video_thumbnail",
		"mcp.port",
	} {
		if !strings.Contains(msg, needle) {
			t.Fatalf("expected error to contain %q, got %q", needle, msg)
		}
	}
}

func TestMergeNormalizesExtensionsAndCompatState(t *testing.T) {
	t.Parallel()

	cfg := Merge(Default(), Overlay{
		Scan: ScanOverlay{
			IncludeExt: strSlicePtr([]string{" JPG ", "mp4"}),
			ExcludeExt: strSlicePtr([]string{" .GIF "}),
		},
		Upload: UploadOverlay{
			StatePathCompat: strPtr("/tmp/legacy-state.sqlite"),
		},
	})

	if got := cfg.Scan.IncludeExt; len(got) != 2 || got[0] != ".jpg" || got[1] != ".mp4" {
		t.Fatalf("unexpected include extensions: %#v", got)
	}
	if got := cfg.Scan.ExcludeExt; len(got) != 1 || got[0] != ".gif" {
		t.Fatalf("unexpected exclude extensions: %#v", got)
	}
	if cfg.Paths.StatePath != "/tmp/legacy-state.sqlite" {
		t.Fatalf("expected upload.state compat to map into paths.state, got %q", cfg.Paths.StatePath)
	}
}

func strPtr(v string) *string { return &v }

func strSlicePtr(v []string) *[]string { return &v }

func osWriteFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0o644)
}

func assertStringPtr(t *testing.T, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected string pointer value %q, got nil", want)
	}
	if *got != want {
		t.Fatalf("expected %q, got %q", want, *got)
	}
}
