package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/babywbx/tgup/internal/config"
	"github.com/babywbx/tgup/internal/media"
	"github.com/babywbx/tgup/internal/plan"
	"github.com/babywbx/tgup/internal/queue"
	"github.com/babywbx/tgup/internal/scan"
	"github.com/babywbx/tgup/internal/state"
	"github.com/babywbx/tgup/internal/tg"
	"github.com/babywbx/tgup/internal/upload"
)

// RunOptions holds parameters for the run command.
type RunOptions struct {
	NoProgress        bool
	ShowPlan          bool
	ShowPlanFiles     bool
	CleanupNow        bool
	ForceMultiCommand bool
	Stdout            io.Writer
	Stderr            io.Writer
}

// RunUpload executes the full upload pipeline and returns an exit code.
// 0 = success, 1 = partial failure, 2 = fatal error.
func RunUpload(configPath string, cli config.Overlay, opts RunOptions) int {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	cfg, err := config.Resolve(configPath, cli)
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}

	// Scan media.
	items, err := scan.Discover(scan.Options{
		Src:            cfg.Scan.Src,
		Recursive:      cfg.Scan.Recursive,
		FollowSymlinks: cfg.Scan.FollowSymlinks,
		IncludeExt:     cfg.Scan.IncludeExt,
		ExcludeExt:     cfg.Scan.ExcludeExt,
	})
	if err != nil {
		fmt.Fprintf(stderr, "scan: %v\n", err)
		return 2
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout, "no media files found")
		return 0
	}

	// Build plan.
	pl := plan.Build(items, plan.Options{
		Order:    cfg.Plan.Order,
		Reverse:  cfg.Plan.Reverse,
		AlbumMax: cfg.Plan.AlbumMax,
	})

	// Show plan preview if requested.
	if opts.ShowPlan || opts.ShowPlanFiles {
		printPlanPreview(stdout, pl, opts.ShowPlanFiles)
	}

	// Resolve paths for state/session (force-multi-command isolation).
	statePath := cfg.Paths.StatePath
	sessionPath := cfg.Telegram.SessionPath
	if opts.ForceMultiCommand {
		suffix := fmt.Sprintf("force.%d.%d", os.Getpid(), time.Now().UnixMilli())
		statePath = strings.TrimSuffix(statePath, filepath.Ext(statePath)) + "." + suffix + ".sqlite"
		// Copy original session to isolated path so auth is preserved.
		forcedSession := strings.TrimSuffix(sessionPath, filepath.Ext(sessionPath)) + "." + suffix + ".session"
		if err := copyFile(sessionPath, forcedSession); err == nil {
			sessionPath = forcedSession
		}
		// If copy fails, keep original session path (shared but functional).
	}

	// Open state store.
	store, err := state.OpenSQLite(statePath)
	if err != nil {
		fmt.Fprintf(stderr, "state: %v\n", err)
		return 2
	}
	defer store.Close()

	// Pre-upload maintenance.
	if cfg.Maintenance.Enabled && !opts.ForceMultiCommand {
		runMaintenance(store, cfg.Maintenance, opts.CleanupNow)
	}

	// Signal handling.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Queue coordination (skip in force-multi-command mode).
	var coord queue.Coordinator
	if !opts.ForceMultiCommand {
		runID := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), os.Getpid())
		coord, err = queue.OpenSQLite(statePath, runID, queue.SQLiteOptions{
			HeartbeatTTL: 30 * time.Second,
			PollInterval: 500 * time.Millisecond,
		})
		if err != nil {
			fmt.Fprintf(stderr, "queue: %v\n", err)
			return 2
		}
		defer coord.Close()

		// Start heartbeat before wait so we're not marked stale while waiting.
		hbCtx, hbCancel := context.WithCancel(ctx)
		defer hbCancel()
		go queue.StartHeartbeat(hbCtx, coord, 5*time.Second, nil)

		fmt.Fprintln(stdout, "waiting for queue turn...")
		if err := coord.WaitUntilTurn(ctx, func(ahead int) {
			if ahead > 0 {
				fmt.Fprintf(stdout, "queue position: %d ahead\n", ahead)
			}
		}); err != nil {
			// Mark as canceled on queue wait failure.
			_ = coord.Cancel(context.Background())
			fmt.Fprintf(stderr, "queue wait: %v\n", err)
			return 2
		}
	}

	// Connect to Telegram (after queue wait to avoid holding connection while waiting).
	client := tg.NewGotdClient(tg.GotdConfig{
		AppID:         cfg.Telegram.APIID,
		AppHash:       cfg.Telegram.APIHash,
		SessionPath:   sessionPath,
		UploadThreads: cfg.Upload.Threads,
	})
	if err := client.Connect(ctx); err != nil {
		if coord != nil {
			_ = coord.Finish(context.Background(), "failed")
		}
		fmt.Fprintf(stderr, "connect: %v\n", err)
		return 2
	}
	defer client.Close(context.Background())

	if !client.IsAuthenticated(ctx) {
		if coord != nil {
			_ = coord.Finish(context.Background(), "failed")
		}
		fmt.Fprintf(stderr, "not authenticated (session=%s): run 'tgup login' first\n", sessionPath)
		return 2
	}

	// Run upload pipeline.
	summary, uploadErr := upload.Run(ctx, upload.Input{
		Plan:        pl.Albums,
		Transport:   client,
		Store:       store,
		Prober:      media.NewChainProber(media.FFProbeMetadataProber{}, media.NativeMetadataProber{}),
		Thumbnailer: media.FFMpegThumbnailer{},
		Config: upload.Config{
			Target:         cfg.Upload.Target,
			Caption:        cfg.Upload.Caption,
			ParseMode:      cfg.Upload.ParseMode,
			Concurrency:    cfg.Upload.ConcurrencyAlbum,
			StrictMetadata: cfg.Upload.StrictMetadata,
			ImageMode:      cfg.Upload.ImageMode,
			VideoThumbnail: cfg.Upload.VideoThumbnail,
			Resume:         cfg.Upload.Resume,
			Duplicate:      upload.DuplicatePolicy(cfg.Upload.Duplicate),
		},
		OnProgress: makeProgressCallback(stdout, opts.NoProgress),
		OnEvent: func(e upload.Event) {
			if e.Error != "" {
				fmt.Fprintf(stderr, "[%s] %s: %s\n", e.Type, e.Album, e.Error)
			}
		},
	})

	// Finish queue with correct status names matching cleanup expectations.
	if coord != nil {
		status := "finished"
		if uploadErr != nil || summary.Failed > 0 {
			status = "failed"
		}
		if summary.Canceled {
			status = "canceled"
		}
		_ = coord.Finish(context.Background(), status)
	}

	if uploadErr != nil {
		fmt.Fprintf(stderr, "upload: %v\n", uploadErr)
		return 2
	}

	// Post-upload maintenance.
	if cfg.Maintenance.Enabled && !opts.ForceMultiCommand {
		runMaintenance(store, cfg.Maintenance, false)
	}

	// Print summary.
	fmt.Fprintf(stdout, "\ndone: sent=%d failed=%d skipped=%d total=%d\n",
		summary.Sent, summary.Failed, summary.Skipped, summary.Total)

	if summary.Canceled {
		return 1
	}
	if summary.Failed > 0 {
		return 1
	}
	return 0
}

func printPlanPreview(w io.Writer, pl plan.Plan, showFiles bool) {
	fmt.Fprintf(w, "plan: %d albums, %d files\n", len(pl.Albums), countPlanFiles(pl))
	for i, album := range pl.Albums {
		fmt.Fprintf(w, "  %d. %s [%d files]\n", i+1, album.Label, len(album.Items))
		if showFiles {
			for _, item := range album.Items {
				fmt.Fprintf(w, "     - %s\n", item.Path)
			}
		}
	}
	fmt.Fprintln(w)
}

func countPlanFiles(pl plan.Plan) int {
	n := 0
	for _, a := range pl.Albums {
		n += len(a.Items)
	}
	return n
}

func runMaintenance(store state.Store, cfg config.MaintenanceConfig, force bool) {
	mainCfg := state.MaintenanceConfig{
		Enabled:    cfg.Enabled,
		MaxAge:     time.Duration(cfg.RetentionSentDays) * 24 * time.Hour,
		KeepFailed: cfg.RetentionFailedDays > 0,
	}
	_, _ = store.ApplyMaintenance(context.Background(), mainCfg, force)
}

func makeProgressCallback(stdout io.Writer, noProgress bool) func(upload.Snapshot) {
	if noProgress {
		return nil
	}
	return func(s upload.Snapshot) {
		if s.TotalBytes <= 0 {
			return
		}
		pct := float64(s.SentBytes) / float64(s.TotalBytes) * 100
		fmt.Fprintf(stdout, "\r[%.1f%%] albums=%d/%d files=%d/%d failed=%d %s",
			pct, s.SentAlbums, s.TotalAlbums, s.SentFiles, s.TotalFiles, s.FailedAlbums, s.CurrentLabel)
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}
