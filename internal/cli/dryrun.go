package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wbx/tgup/internal/app"
	"github.com/wbx/tgup/internal/artifacts"
	"github.com/wbx/tgup/internal/config"
)

func runDryRun(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	var reportDir string
	fs.StringVar(&configPath, "config", "", "path to config file")
	fs.StringVar(&reportDir, "report-dir", "", "write report.json/report.md into this directory")

	src := &csvValue{}
	includeExt := &csvValue{}
	excludeExt := &csvValue{}
	order := &stringValue{}
	target := &stringValue{}
	parseMode := &stringValue{}
	duplicate := &stringValue{}
	imageMode := &stringValue{}
	sessionPath := &stringValue{}
	sessionPathAlias := &stringValue{}
	statePath := &stringValue{}
	statePathAlias := &stringValue{}
	artifactsDir := &stringValue{}
	videoThumbnail := &stringValue{}
	caption := &stringValue{}

	recursive := &boolValue{}
	followSymlinks := &boolValue{}
	reverse := &boolValue{}
	resume := &boolValue{}
	strictMetadata := &boolValue{}
	maintenanceEnabled := &boolValue{}
	mcpEnabled := &boolValue{}

	albumMax := &intValue{}
	concurrencyAlbum := &intValue{}
	apiID := &intValue{}
	mcpPort := &intValue{}
	apiHash := &stringValue{}
	mcpHost := &stringValue{}

	fs.Var(src, "src", "source path(s), repeatable or comma-separated")
	fs.Var(includeExt, "include-ext", "include extension(s), repeatable or comma-separated")
	fs.Var(excludeExt, "exclude-ext", "exclude extension(s), repeatable or comma-separated")
	fs.Var(order, "order", "plan order: name|mtime|size|random")
	fs.Var(target, "target", "telegram target")
	fs.Var(parseMode, "parse-mode", "caption parse mode")
	fs.Var(duplicate, "duplicate", "duplicate policy: skip|ask|upload")
	fs.Var(imageMode, "image-mode", "image mode: auto|photo|document")
	fs.Var(sessionPath, "session", "session file path")
	fs.Var(sessionPathAlias, "session-path", "session file path (alias)")
	fs.Var(statePath, "state", "state sqlite path")
	fs.Var(statePathAlias, "state-path", "state sqlite path (alias)")
	fs.Var(artifactsDir, "artifacts-dir", "artifacts output directory")
	fs.Var(videoThumbnail, "video-thumbnail", "video thumbnail path or auto")
	fs.Var(caption, "caption", "default caption")
	fs.Var(apiHash, "api-hash", "telegram api hash")
	fs.Var(mcpHost, "mcp-host", "mcp host")

	fs.Var(recursive, "recursive", "scan recursively")
	fs.Var(followSymlinks, "follow-symlinks", "follow symlink paths")
	fs.Var(reverse, "reverse", "reverse order")
	fs.Var(resume, "resume", "enable resume")
	fs.Var(strictMetadata, "strict-metadata", "strict metadata checks")
	fs.Var(maintenanceEnabled, "maintenance-enabled", "enable maintenance")
	fs.Var(mcpEnabled, "mcp-enabled", "enable mcp")

	fs.Var(albumMax, "album-max", "max media per album")
	fs.Var(concurrencyAlbum, "concurrency-album", "album concurrency")
	fs.Var(apiID, "api-id", "telegram api id")
	fs.Var(mcpPort, "mcp-port", "mcp port")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected dry-run args: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}
	if sessionPath.set && sessionPathAlias.set {
		fmt.Fprintln(stderr, "conflicting flags: --session and --session-path")
		return 2
	}
	if sessionPathAlias.set && !sessionPath.set {
		sessionPath = sessionPathAlias
	}
	if statePath.set && statePathAlias.set {
		fmt.Fprintln(stderr, "conflicting flags: --state and --state-path")
		return 2
	}
	if statePathAlias.set && !statePath.set {
		statePath = statePathAlias
	}

	cli := config.Overlay{}
	if src.set {
		cli.Scan.Src = src.ptr()
	}
	if includeExt.set {
		cli.Scan.IncludeExt = includeExt.ptr()
	}
	if excludeExt.set {
		cli.Scan.ExcludeExt = excludeExt.ptr()
	}
	if order.set {
		cli.Plan.Order = order.ptr()
	}
	if target.set {
		cli.Upload.Target = target.ptr()
	}
	if parseMode.set {
		cli.Upload.ParseMode = parseMode.ptr()
	}
	if duplicate.set {
		cli.Upload.Duplicate = duplicate.ptr()
	}
	if imageMode.set {
		cli.Upload.ImageMode = imageMode.ptr()
	}
	if sessionPath.set {
		cli.Telegram.SessionPath = sessionPath.ptr()
	}
	if statePath.set {
		cli.Paths.StatePath = statePath.ptr()
	}
	if artifactsDir.set {
		cli.Paths.ArtifactsDir = artifactsDir.ptr()
	}
	if videoThumbnail.set {
		cli.Upload.VideoThumbnail = videoThumbnail.ptr()
	}
	if caption.set {
		cli.Upload.Caption = caption.ptr()
	}
	if apiHash.set {
		cli.Telegram.APIHash = apiHash.ptr()
	}
	if mcpHost.set {
		cli.MCP.Host = mcpHost.ptr()
	}
	if recursive.set {
		cli.Scan.Recursive = recursive.ptr()
	}
	if followSymlinks.set {
		cli.Scan.FollowSymlinks = followSymlinks.ptr()
	}
	if reverse.set {
		cli.Plan.Reverse = reverse.ptr()
	}
	if resume.set {
		cli.Upload.Resume = resume.ptr()
	}
	if strictMetadata.set {
		cli.Upload.StrictMetadata = strictMetadata.ptr()
	}
	if maintenanceEnabled.set {
		cli.Maintenance.Enabled = maintenanceEnabled.ptr()
	}
	if mcpEnabled.set {
		cli.MCP.Enabled = mcpEnabled.ptr()
	}
	if albumMax.set {
		cli.Plan.AlbumMax = albumMax.ptr()
	}
	if concurrencyAlbum.set {
		cli.Upload.ConcurrencyAlbum = concurrencyAlbum.ptr()
	}
	if apiID.set {
		cli.Telegram.APIID = apiID.ptr()
	}
	if mcpPort.set {
		cli.MCP.Port = mcpPort.ptr()
	}

	result, err := app.ExecuteDryRun(configPath, cli)
	if err != nil {
		fmt.Fprintf(stderr, "dry-run failed: %v\n", err)
		return 1
	}
	if err := app.WriteDryRun(stdout, result, app.RenderOptions{
		MaxAlbums:        5,
		MaxItemsPerAlbum: 3,
	}); err != nil {
		fmt.Fprintf(stderr, "dry-run failed to write output: %v\n", err)
		return 1
	}

	if strings.TrimSpace(reportDir) != "" {
		writer := artifacts.NewFileReportWriter(reportDir)
		summary := app.BuildDryRunSummary(result, app.RenderOptions{
			MaxAlbums:        5,
			MaxItemsPerAlbum: 3,
		})
		if err := writer.WriteJSON(summary); err != nil {
			fmt.Fprintf(stderr, "dry-run failed to write report json: %v\n", err)
			return 1
		}
		if err := writer.WriteMarkdown(summary); err != nil {
			fmt.Fprintf(stderr, "dry-run failed to write report markdown: %v\n", err)
			return 1
		}
		if _, err := fmt.Fprintf(stdout, "\nreports\njson: %s\nmarkdown: %s\n", writer.JSONPath, writer.MarkdownPath); err != nil {
			fmt.Fprintf(stderr, "dry-run failed to write output: %v\n", err)
			return 1
		}
	}
	return 0
}

type stringValue struct {
	set   bool
	value string
}

func (v *stringValue) String() string { return v.value }

func (v *stringValue) Set(s string) error {
	v.value = s
	v.set = true
	return nil
}

func (v *stringValue) ptr() *string { return &v.value }

type boolValue struct {
	set   bool
	value bool
}

func (v *boolValue) String() string { return strconv.FormatBool(v.value) }

func (v *boolValue) IsBoolFlag() bool { return true }

func (v *boolValue) Set(s string) error {
	parsed, err := strconv.ParseBool(s)
	if err != nil {
		return errors.New("must be true or false")
	}
	v.value = parsed
	v.set = true
	return nil
}

func (v *boolValue) ptr() *bool { return &v.value }

type intValue struct {
	set   bool
	value int
}

func (v *intValue) String() string { return strconv.Itoa(v.value) }

func (v *intValue) Set(s string) error {
	parsed, err := strconv.Atoi(s)
	if err != nil {
		return errors.New("must be an integer")
	}
	v.value = parsed
	v.set = true
	return nil
}

func (v *intValue) ptr() *int { return &v.value }

type csvValue struct {
	set    bool
	values []string
}

func (v *csvValue) String() string {
	return strings.Join(v.values, ",")
}

func (v *csvValue) Set(s string) error {
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v.values = append(v.values, part)
	}
	v.set = true
	return nil
}

func (v *csvValue) ptr() *[]string {
	values := make([]string, len(v.values))
	copy(values, v.values)
	return &values
}
