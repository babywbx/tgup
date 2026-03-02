package media

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"os"
	"os/exec"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
)

// Telegram sendPhoto hard constraints.
const (
	detectSumWH    = 10000     // trigger compression when w+h > this
	targetSumWH    = 9600      // compression target (empirically safe max)
	maxAspectRatio = 20        // max(w,h) / min(w,h) ≤ 20
	maxPhotoBytes  = 9_500_000 // 9.5 MB, safe margin below TG's ~10 MB limit
	fallbackEdge   = 2560      // long edge for last-resort compression
)

type scaleMode int

const (
	scaleNone     scaleMode = iota // no resize, re-encode only
	scaleSumWH                     // scale to w+h ≤ targetSumWH
	scaleLongEdge                  // scale to long edge ≤ fallbackEdge
)

// CompressPhoto checks if an image exceeds Telegram photo limits and
// returns a compressed JPEG path if needed.
// Returns ("", nil, nil) when no compression is required.
// Uses ffmpeg (Lanczos) when available, falls back to Go stdlib (CatmullRom).
// Quality is maximized via binary search to fill as close to 9.5 MB as possible.
func CompressPhoto(path string) (compressedPath string, cleanup func(), err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("open image: %w", err)
	}

	cfg, format, err := image.DecodeConfig(f)
	f.Close()
	if err != nil {
		return "", nil, nil
	}

	if format != "jpeg" && format != "png" {
		return "", nil, nil
	}

	fi, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("stat image: %w", err)
	}

	if !needsCompress(cfg.Width, cfg.Height, fi.Size(), detectSumWH) {
		return "", nil, nil
	}

	// Aspect ratio > 20 can't be sent as photo regardless.
	if aspectRatio(cfg.Width, cfg.Height) > maxAspectRatio {
		return "", nil, nil
	}

	_, lookErr := exec.LookPath("ffmpeg")
	hasFFmpeg := lookErr == nil

	// Build mode list: size-only → try original dims first;
	// dimension violation → must scale down.
	dimOK := cfg.Width+cfg.Height <= detectSumWH
	var modes []scaleMode
	if dimOK {
		modes = []scaleMode{scaleNone, scaleSumWH, scaleLongEdge}
	} else {
		modes = []scaleMode{scaleSumWH, scaleLongEdge}
	}

	for _, mode := range modes {
		// scaleNone: prefer stdlib (finer quality granularity fills space better).
		// resize modes: prefer ffmpeg Lanczos (superior scaling quality).
		if mode == scaleNone || !hasFFmpeg {
			compressedPath, cleanup, err = searchBestStdlibQ(path, format, mode)
		} else {
			compressedPath, cleanup, err = searchBestFFmpegQ(path, mode)
		}
		if err == nil {
			return compressedPath, cleanup, nil
		}
	}

	return "", nil, fmt.Errorf("image %s: cannot compress below %d bytes", path, maxPhotoBytes)
}

// needsCompress reports whether the image exceeds Telegram photo limits.
func needsCompress(w, h int, size int64, maxSumWH int) bool {
	return w+h > maxSumWH || size > maxPhotoBytes
}

// aspectRatio returns max(w,h)/min(w,h).
func aspectRatio(w, h int) float64 {
	if w <= 0 || h <= 0 {
		return 0
	}
	if w >= h {
		return float64(w) / float64(h)
	}
	return float64(h) / float64(w)
}

// fitSumWH returns dimensions scaled so w+h ≤ maxSum, preserving aspect ratio.
func fitSumWH(w, h, maxSum int) (int, int) {
	if w+h <= maxSum {
		return w, h
	}
	scale := float64(maxSum) / float64(w+h)
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return nw, nh
}

// fitLongEdge returns dimensions scaled so max(w,h) ≤ maxEdge.
func fitLongEdge(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		return maxEdge, max(h*maxEdge/w, 1)
	}
	return max(w*maxEdge/h, 1), maxEdge
}

// --- ffmpeg backend ---
// Uses filter expressions referencing post-rotation iw/ih, so EXIF
// orientation is handled correctly (ffmpeg auto-rotates before filters).

func ffmpegScaleFilter(mode scaleMode) string {
	switch mode {
	case scaleNone:
		return ""
	case scaleSumWH:
		return fmt.Sprintf(
			"scale='trunc(iw*min(1\\,%d/(iw+ih)))':'trunc(ih*min(1\\,%d/(iw+ih)))':flags=lanczos",
			targetSumWH, targetSumWH,
		)
	case scaleLongEdge:
		return fmt.Sprintf(
			"scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease:flags=lanczos",
			fallbackEdge, fallbackEdge,
		)
	default:
		return ""
	}
}

// searchBestFFmpegQ binary-searches ffmpeg -q:v (1=best, 31=worst) to find
// the highest quality whose output fits within maxPhotoBytes.
func searchBestFFmpegQ(path string, mode scaleMode) (string, func(), error) {
	filter := ffmpegScaleFilter(mode)

	// Quick check: if Q1 (best) already fits, no search needed.
	bestPath, bestCleanup, err := runFFmpeg(path, filter, 1)
	if err != nil {
		return "", nil, err
	}
	if fileSize(bestPath) <= maxPhotoBytes {
		return bestPath, bestCleanup, nil
	}
	bestCleanup()

	// Binary search between 2 and 31.
	var resultPath string
	var resultCleanup func()
	low, high := 2, 31
	for low <= high {
		mid := (low + high) / 2
		p, cl, err := runFFmpeg(path, filter, mid)
		if err != nil {
			if resultCleanup != nil {
				resultCleanup()
			}
			return "", nil, err
		}
		if fileSize(p) <= maxPhotoBytes {
			if resultCleanup != nil {
				resultCleanup()
			}
			resultPath = p
			resultCleanup = cl
			high = mid - 1 // try better quality
		} else {
			cl()
			low = mid + 1 // need lower quality
		}
	}

	if resultPath != "" {
		return resultPath, resultCleanup, nil
	}
	return "", nil, fmt.Errorf("ffmpeg: cannot fit %s within %d bytes at mode %d", path, maxPhotoBytes, mode)
}

func runFFmpeg(path, scaleFilter string, qv int) (string, func(), error) {
	tmp, err := os.CreateTemp("", "tgup-compress-*.jpg")
	if err != nil {
		return "", nil, fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()

	cleanupFn := func() { _ = os.Remove(tmpPath) }

	args := []string{
		"-y", "-loglevel", "error",
		"-i", path,
	}
	if scaleFilter != "" {
		args = append(args, "-vf", scaleFilter)
	}
	args = append(args,
		"-c:v", "mjpeg",
		"-huffman", "optimal",
		"-map_metadata", "-1",
		"-q:v", fmt.Sprint(qv),
		tmpPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		cleanupFn()
		return "", nil, fmt.Errorf("ffmpeg: %s: %w", string(out), runErr)
	}

	info, err := os.Stat(tmpPath)
	if err != nil || info.Size() == 0 {
		cleanupFn()
		return "", nil, fmt.Errorf("ffmpeg produced empty output for %s", path)
	}

	return tmpPath, cleanupFn, nil
}

// --- Go stdlib backend ---

// searchBestStdlibQ decodes once, then binary-searches JPEG quality (99=best, 1=worst)
// to find the highest quality whose output fits within maxPhotoBytes.
func searchBestStdlibQ(path string, format string, mode scaleMode) (string, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return "", nil, fmt.Errorf("decode image: %w", err)
	}

	// Apply EXIF orientation BEFORE computing target dimensions.
	if format == "jpeg" {
		if _, seekErr := f.Seek(0, io.SeekStart); seekErr == nil {
			src = applyExifOrientation(f, src)
		}
	}

	// Compute target from actual (post-rotation) dimensions.
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	switch mode {
	case scaleSumWH:
		tw, th := fitSumWH(w, h, targetSumWH)
		if tw != w || th != h {
			src = resizeImage(src, tw, th)
		}
	case scaleLongEdge:
		tw, th := fitLongEdge(w, h, fallbackEdge)
		if tw != w || th != h {
			src = resizeImage(src, tw, th)
		}
	case scaleNone:
		// keep original dimensions
	}

	// Quick check: if Q100 (best) already fits, no search needed.
	bestPath, bestCleanup, err := encodeJPEG(src, 100)
	if err != nil {
		return "", nil, err
	}
	if fileSize(bestPath) <= maxPhotoBytes {
		return bestPath, bestCleanup, nil
	}
	bestCleanup()

	// Binary search between 1 and 99.
	var resultPath string
	var resultCleanup func()
	low, high := 1, 99
	for low <= high {
		mid := (low + high + 1) / 2 // round up to prefer higher quality
		p, cl, err := encodeJPEG(src, mid)
		if err != nil {
			if resultCleanup != nil {
				resultCleanup()
			}
			return "", nil, err
		}
		if fileSize(p) <= maxPhotoBytes {
			if resultCleanup != nil {
				resultCleanup()
			}
			resultPath = p
			resultCleanup = cl
			low = mid + 1 // try better quality
		} else {
			cl()
			high = mid - 1 // need lower quality
		}
	}

	if resultPath != "" {
		return resultPath, resultCleanup, nil
	}
	return "", nil, fmt.Errorf("stdlib: cannot fit within %d bytes at mode %d", maxPhotoBytes, mode)
}

func encodeJPEG(src image.Image, quality int) (string, func(), error) {
	tmp, err := os.CreateTemp("", "tgup-compress-*.jpg")
	if err != nil {
		return "", nil, fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupFn := func() { _ = os.Remove(tmpPath) }

	if err := jpeg.Encode(tmp, src, &jpeg.Options{Quality: quality}); err != nil {
		tmp.Close()
		cleanupFn()
		return "", nil, fmt.Errorf("encode jpeg: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanupFn()
		return "", nil, fmt.Errorf("close temp: %w", err)
	}
	return tmpPath, cleanupFn, nil
}

// fileSize returns file size or 0 on error.
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// resizeImage scales src to exact tw×th using CatmullRom interpolation.
func resizeImage(src image.Image, tw, th int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// --- EXIF orientation (stdlib fallback) ---

func applyExifOrientation(r io.Reader, img image.Image) image.Image {
	x, err := exif.Decode(r)
	if err != nil {
		return img
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return img
	}
	orient, err := tag.Int(0)
	if err != nil {
		return img
	}

	switch orient {
	case 2:
		return flipH(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipV(img)
	case 5: // transpose: rotate 90 CW then flip horizontal
		return flipH(rotate90(img))
	case 6:
		return rotate90(img)
	case 7: // transverse: rotate 270 CW then flip horizontal
		return flipH(rotate270(img))
	case 8:
		return rotate270(img)
	default:
		return img
	}
}

func rotate90(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.Y-1-y, x, img.At(x, y))
		}
	}
	return dst
}

func rotate180(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x, b.Max.Y-1-y, img.At(x, y))
		}
	}
	return dst
}

func rotate270(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(y, b.Max.X-1-x, img.At(x, y))
		}
	}
	return dst
}

func flipH(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x, y, img.At(x, y))
		}
	}
	return dst
}

func flipV(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, b.Max.Y-1-y, img.At(x, y))
		}
	}
	return dst
}
