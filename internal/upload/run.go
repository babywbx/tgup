package upload

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/babywbx/tgup/internal/media"
	"github.com/babywbx/tgup/internal/plan"
	"github.com/babywbx/tgup/internal/scan"
	"github.com/babywbx/tgup/internal/state"
	"github.com/babywbx/tgup/internal/tg"
)

const (
	maxRetries    = 5
	maxFloodWaits = 20
	maxBackoff    = 60 * time.Second
)

// Run executes the full upload pipeline: precheck, send, postcheck, state mark.
func Run(ctx context.Context, in Input) (Summary, error) {
	if in.Config.Concurrency <= 0 {
		in.Config.Concurrency = 5
	}

	albums := in.Plan
	totalFiles := 0
	var totalBytes int64
	for _, a := range albums {
		totalFiles += len(a.Items)
		for _, item := range a.Items {
			totalBytes += item.Size
		}
	}

	summary := Summary{Total: totalFiles}
	if totalFiles == 0 {
		return summary, nil
	}

	if in.Transport == nil {
		return summary, fmt.Errorf("transport is required")
	}

	// Resolve target once.
	target, err := in.Transport.ResolveTarget(ctx, in.Config.Target)
	if err != nil {
		return summary, fmt.Errorf("resolve target: %w", err)
	}

	// Worker pool.
	work := make(chan indexedAlbum, len(albums))
	for i, a := range albums {
		work <- indexedAlbum{index: i, album: a}
	}
	close(work)

	var (
		sentFiles    atomic.Int64
		failedFiles  atomic.Int64
		skippedFiles atomic.Int64
		sentBytes    atomic.Int64
		sentAlbums   atomic.Int64
		failedAlbums atomic.Int64
		canceled     atomic.Bool
	)

	workers := in.Config.Concurrency
	if workers > len(albums) {
		workers = len(albums)
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ia := range work {
				if ctx.Err() != nil {
					canceled.Store(true)
					return
				}
				processAlbum(ctx, ia, processArgs{
					input:        in,
					target:       target,
					totalFiles:   totalFiles,
					totalBytes:   totalBytes,
					totalAlbums:  len(albums),
					sentFiles:    &sentFiles,
					failedFiles:  &failedFiles,
					skippedFiles: &skippedFiles,
					sentBytes:    &sentBytes,
					sentAlbums:   &sentAlbums,
					failedAlbums: &failedAlbums,
				})
			}
		}()
	}
	wg.Wait()

	if ctx.Err() != nil {
		canceled.Store(true)
	}

	summary.Sent = int(sentFiles.Load())
	summary.Failed = int(failedFiles.Load())
	summary.Skipped = int(skippedFiles.Load())
	summary.Canceled = canceled.Load()
	return summary, nil
}

type indexedAlbum struct {
	index int
	album plan.Album
}

type processArgs struct {
	input        Input
	target       tg.ResolvedTarget
	totalFiles   int
	totalBytes   int64
	totalAlbums  int
	sentFiles    *atomic.Int64
	failedFiles  *atomic.Int64
	skippedFiles *atomic.Int64
	sentBytes    *atomic.Int64
	sentAlbums   *atomic.Int64
	failedAlbums *atomic.Int64
}

func processAlbum(ctx context.Context, ia indexedAlbum, args processArgs) {
	album := ia.album
	in := args.input
	files := toAlbumFiles(album.Items)

	emitEvent(in, Event{Type: "upload.album.start", Album: album.Label, Files: len(files)})
	emitProgress(in, args, album.Label)

	// 1. Resume filtering per file.
	if in.Config.Resume && in.Store != nil {
		var pending []AlbumFile
		for _, f := range files {
			done, err := in.Store.IsDone(ctx, ResumeKeyFromFile(f))
			if err != nil {
				// State error — include file anyway.
				pending = append(pending, f)
				continue
			}
			if done {
				switch in.Config.Duplicate {
				case DuplicateSkip:
					args.skippedFiles.Add(1)
					args.sentBytes.Add(f.Size)
					continue
				case DuplicateUpload:
					pending = append(pending, f)
				default: // "ask" — skip in non-interactive mode
					args.skippedFiles.Add(1)
					args.sentBytes.Add(f.Size)
					continue
				}
			} else {
				pending = append(pending, f)
			}
		}
		files = pending
	}

	if len(files) == 0 {
		args.sentAlbums.Add(1)
		emitEvent(in, Event{Type: "upload.album.done", Album: album.Label, Files: 0})
		return
	}

	// 2. Precheck video metadata.
	precheck := PrecheckAlbum(ctx, in.Prober, files)
	if len(precheck.Warnings) > 0 {
		emitEvent(in, Event{
			Type:  "upload.album.precheck_warning",
			Album: album.Label,
			Error: strings.Join(precheck.Warnings, "; "),
		})
	}
	if precheck.HasViolations() && in.Config.StrictMetadata {
		for _, f := range files {
			markFailed(ctx, in, f, args.target.Raw, "precheck: metadata violation")
		}
		args.failedFiles.Add(int64(len(files)))
		args.failedAlbums.Add(1)
		emitEvent(in, Event{Type: "upload.album.failed", Album: album.Label, Error: "metadata violation"})
		return
	}

	// 3. Generate thumbnails.
	type thumbInfo struct {
		path    string
		cleanup func()
	}
	thumbs := make(map[int]thumbInfo)
	videoThumbPolicy := strings.ToLower(strings.TrimSpace(in.Config.VideoThumbnail))
	switch videoThumbPolicy {
	case "auto":
		if in.Thumbnailer != nil {
			for i, f := range files {
				if f.Kind != media.KindVideo {
					continue
				}
				tp, cleanup, err := in.Thumbnailer.ExtractVideoThumbnail(ctx, f.Path)
				if err != nil {
					emitEvent(in, Event{
						Type:  "upload.album.thumbnail_warning",
						Album: album.Label,
						Error: fmt.Sprintf("%s: %v", f.Path, err),
					})
					continue
				}
				thumbs[i] = thumbInfo{path: tp, cleanup: cleanup}
				files[i].Thumbnail = tp
			}
		}
	case "", "off":
		// Thumbnail disabled.
	default:
		// Explicit path — use as thumbnail for all videos.
		for i, f := range files {
			if f.Kind == media.KindVideo {
				files[i].Thumbnail = in.Config.VideoThumbnail
			}
		}
	}
	defer func() {
		for _, ti := range thumbs {
			if ti.cleanup != nil {
				ti.cleanup()
			}
		}
	}()

	// 4. Send with retry.
	sendResult, sendErr := sendWithRetry(ctx, in, files, args.target)
	if sendErr != nil {
		for _, f := range files {
			markFailed(ctx, in, f, args.target.Raw, sendErr.Error())
		}
		args.failedFiles.Add(int64(len(files)))
		args.failedAlbums.Add(1)
		emitEvent(in, Event{Type: "upload.album.failed", Album: album.Label, Error: sendErr.Error()})
		return
	}

	// 5. Postcheck.
	pc := PostcheckMessages(sendResult, len(files))
	if !pc.OK {
		emitEvent(in, Event{
			Type:  "upload.album.postcheck_warning",
			Album: album.Label,
			Error: strings.Join(pc.Issues, "; "),
		})
	}

	// 6. Mark sent in state.
	for i, f := range files {
		var msgIDs []int
		if i < len(sendResult.MessageIDs) && sendResult.MessageIDs[i] > 0 {
			msgIDs = []int{sendResult.MessageIDs[i]}
		} else if i < len(sendResult.Messages) && sendResult.Messages[i].ID > 0 {
			msgIDs = []int{sendResult.Messages[i].ID}
		}
		markSent(ctx, in, f, args.target.Raw, msgIDs, sendResult.GroupID)
	}

	var albumBytes int64
	for _, f := range files {
		albumBytes += f.Size
	}
	args.sentFiles.Add(int64(len(files)))
	args.sentBytes.Add(albumBytes)
	args.sentAlbums.Add(1)

	emitEvent(in, Event{Type: "upload.album.done", Album: album.Label, Files: len(files)})
	emitProgress(in, args, "")
}

func sendWithRetry(ctx context.Context, in Input, files []AlbumFile, target tg.ResolvedTarget) (tg.SendResult, error) {
	imageMode := strings.ToLower(strings.TrimSpace(in.Config.ImageMode))
	forceDocument := imageMode == "document"
	retryAttempt := 0
	floodWaitCount := 0

	for {
		if ctx.Err() != nil {
			return tg.SendResult{}, ctx.Err()
		}

		result, err := sendAlbumOnce(ctx, in, files, target, forceDocument)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return tg.SendResult{}, err
		}

		// ImageProcessFailed → retry as document (not counted as retry).
		if tg.IsImageProcessFailed(err) && !forceDocument && imageMode == "auto" {
			forceDocument = true
			continue
		}

		// FloodWait → sleep exact duration (not counted as retry).
		if d := tg.FloodWaitDuration(err); d > 0 {
			floodWaitCount++
			if floodWaitCount > maxFloodWaits {
				return tg.SendResult{}, fmt.Errorf("too many flood waits: %w", err)
			}
			timer := time.NewTimer(d + time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return tg.SendResult{}, ctx.Err()
			case <-timer.C:
			}
			continue
		}

		// Retryable errors → backoff.
		if !tg.IsRetryable(err) {
			return tg.SendResult{}, err
		}
		if retryAttempt >= maxRetries {
			return tg.SendResult{}, fmt.Errorf("max retries exceeded: %w", err)
		}

		delay := backoffWithJitter(retryAttempt)
		retryAttempt++
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return tg.SendResult{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func sendAlbumOnce(ctx context.Context, in Input, files []AlbumFile, target tg.ResolvedTarget, forceDocument bool) (tg.SendResult, error) {
	if len(files) == 1 {
		f := files[0]
		return in.Transport.SendSingle(ctx, tg.SendSingleRequest{
			Target:            target,
			Path:              f.Path,
			Caption:           in.Config.Caption,
			ParseMode:         in.Config.ParseMode,
			ForceDocument:     forceDocument,
			SupportsStreaming: true,
			ThumbnailPath:     f.Thumbnail,
		})
	}

	items := make([]tg.AlbumMedia, 0, len(files))
	for i, f := range files {
		caption := ""
		if i == 0 {
			caption = in.Config.Caption
		}
		items = append(items, tg.AlbumMedia{
			Path:              f.Path,
			Caption:           caption,
			ForceDocument:     forceDocument,
			SupportsStreaming: true,
			ThumbnailPath:     f.Thumbnail,
		})
	}

	return in.Transport.SendAlbum(ctx, tg.SendAlbumRequest{
		Target:    target,
		Items:     items,
		ParseMode: in.Config.ParseMode,
	})
}

func backoffWithJitter(attempt int) time.Duration {
	delay := Backoff(attempt)
	if delay > maxBackoff {
		delay = maxBackoff
	}
	return time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5))
}

func toAlbumFiles(items []scan.Item) []AlbumFile {
	out := make([]AlbumFile, 0, len(items))
	for _, item := range items {
		out = append(out, AlbumFile{
			Path:    item.Path,
			Size:    item.Size,
			MTimeNS: item.MTimeNS,
			Kind:    toMediaKind(item.Kind),
		})
	}
	return out
}

func toMediaKind(kind scan.Kind) media.Kind {
	switch kind {
	case scan.KindImage:
		return media.KindImage
	case scan.KindVideo:
		return media.KindVideo
	default:
		return ""
	}
}

func markSent(ctx context.Context, in Input, f AlbumFile, target string, msgIDs []int, groupID string) {
	if in.Store == nil {
		return
	}
	_ = in.Store.MarkSent(ctx, state.MarkSentInput{
		Key:          ResumeKeyFromFile(f),
		Target:       target,
		MessageIDs:   msgIDs,
		AlbumGroupID: groupID,
	})
}

func markFailed(ctx context.Context, in Input, f AlbumFile, target string, reason string) {
	if in.Store == nil {
		return
	}
	_ = in.Store.MarkFailed(ctx, state.MarkFailedInput{
		Key:         ResumeKeyFromFile(f),
		Target:      target,
		ErrorReason: reason,
	})
}

func emitEvent(in Input, e Event) {
	if in.OnEvent != nil {
		in.OnEvent(e)
	}
}

func emitProgress(in Input, args processArgs, currentLabel string) {
	if in.OnProgress == nil {
		return
	}
	in.OnProgress(Snapshot{
		SentBytes:    args.sentBytes.Load(),
		TotalBytes:   args.totalBytes,
		SentFiles:    int(args.sentFiles.Load()),
		TotalFiles:   args.totalFiles,
		SentAlbums:   int(args.sentAlbums.Load()),
		TotalAlbums:  args.totalAlbums,
		FailedAlbums: int(args.failedAlbums.Load()),
		CurrentLabel: currentLabel,
	})
}
