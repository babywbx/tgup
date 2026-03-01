package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FFProbeMetadataProber probes video metadata via ffprobe CLI.
type FFProbeMetadataProber struct{}

// ProbeVideo extracts duration, width, and height from a video file.
func (FFProbeMetadataProber) ProbeVideo(ctx context.Context, path string) (VideoMetadata, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return VideoMetadata{}, fmt.Errorf("ffprobe_missing: ffprobe not found in PATH")
	}

	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=duration,width,height:format=duration",
		"-of", "json",
		path,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	out, err := cmd.Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return VideoMetadata{}, ctxErr
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return VideoMetadata{}, fmt.Errorf("ffprobe_exit:%d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return VideoMetadata{}, fmt.Errorf("ffprobe_error: %w", err)
	}

	return parseFFProbeJSON(out)
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration string `json:"duration"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

func parseFFProbeJSON(data []byte) (VideoMetadata, error) {
	var probe ffprobeOutput
	if err := json.Unmarshal(data, &probe); err != nil {
		return VideoMetadata{}, fmt.Errorf("ffprobe_invalid_json: %w", err)
	}

	if len(probe.Streams) == 0 {
		return VideoMetadata{}, fmt.Errorf("ffprobe_no_stream: no video stream found")
	}

	stream := probe.Streams[0]
	meta := VideoMetadata{
		Width:  stream.Width,
		Height: stream.Height,
	}

	// Stream-level duration first, fallback to format-level.
	streamDur := strings.TrimSpace(stream.Duration)
	if streamDur != "" {
		if d, err := strconv.ParseFloat(streamDur, 64); err == nil {
			meta.DurationSeconds = d
		}
	}
	fmtDur := strings.TrimSpace(probe.Format.Duration)
	if meta.DurationSeconds == 0 && fmtDur != "" {
		if d, err := strconv.ParseFloat(fmtDur, 64); err == nil {
			meta.DurationSeconds = d
		}
	}

	return meta, nil
}

// CheckFFProbe returns nil if ffprobe is available on PATH.
func CheckFFProbe() error {
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return fmt.Errorf("ffprobe not found in PATH")
	}
	return nil
}
