package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/babywbx/tgup/internal/app"
	"github.com/babywbx/tgup/internal/config"
)

func runRun(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	var noProgress bool
	var showPlan bool
	var showPlanFiles bool
	var cleanupNow bool
	var forceMultiCommand bool

	fs.StringVar(&configPath, "config", "", "path to config file")
	fs.BoolVar(&noProgress, "no-progress", false, "suppress progress output")
	fs.BoolVar(&showPlan, "plan", false, "show plan preview before uploading")
	fs.BoolVar(&showPlanFiles, "plan-files", false, "show plan file details before uploading")
	fs.BoolVar(&cleanupNow, "cleanup-now", false, "force maintenance cleanup before upload")
	fs.BoolVar(&forceMultiCommand, "force-multi-command", false, "isolate session/state for parallel uploads")

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

	albumMax := &intValue{}
	concurrencyAlbum := &intValue{}
	threads := &intValue{}
	apiID := &intValue{}
	apiHash := &stringValue{}

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

	fs.Var(recursive, "recursive", "scan recursively")
	fs.Var(followSymlinks, "follow-symlinks", "follow symlink paths")
	fs.Var(reverse, "reverse", "reverse order")
	fs.Var(resume, "resume", "enable resume")
	fs.Var(strictMetadata, "strict-metadata", "strict metadata checks")

	fs.Var(albumMax, "album-max", "max media per album")
	fs.Var(concurrencyAlbum, "concurrency-album", "album concurrency")
	fs.Var(threads, "threads", "parallel part uploads per file")
	fs.Var(apiID, "api-id", "telegram api id")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected run args: %s\n", strings.Join(fs.Args(), " "))
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
	if albumMax.set {
		cli.Plan.AlbumMax = albumMax.ptr()
	}
	if concurrencyAlbum.set {
		cli.Upload.ConcurrencyAlbum = concurrencyAlbum.ptr()
	}
	if threads.set {
		cli.Upload.Threads = threads.ptr()
	}
	if apiID.set {
		cli.Telegram.APIID = apiID.ptr()
	}

	code := app.RunUpload(configPath, cli, app.RunOptions{
		NoProgress:        noProgress,
		ShowPlan:          showPlan,
		ShowPlanFiles:     showPlanFiles,
		CleanupNow:        cleanupNow,
		ForceMultiCommand: forceMultiCommand,
		Stdout:            stdout,
		Stderr:            stderr,
	})
	return code
}
