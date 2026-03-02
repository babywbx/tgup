package app

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/babywbx/tgup/internal/config"
)

// RunDemo generates test images and uploads them to Saved Messages.
func RunDemo(configPath string, cli config.Overlay, stdout io.Writer) error {
	tempDir, err := os.MkdirTemp("", "tgup-demo-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate two 200x200 solid-color test PNGs.
	for _, tc := range []struct {
		name string
		c    color.Color
	}{
		{"black.png", color.Black},
		{"white.png", color.White},
	} {
		if err := generateTestImage(filepath.Join(tempDir, tc.name), tc.c); err != nil {
			return fmt.Errorf("generate %s: %w", tc.name, err)
		}
	}

	// Build overlay to direct RunUpload at the temp dir → Saved Messages.
	src := []string{tempDir}
	target := "me"
	caption := "Hello from tgup!"
	order := "name"
	albumMax := 2

	cli.Scan.Src = &src
	cli.Upload.Target = &target
	cli.Upload.Caption = &caption
	cli.Plan.Order = &order
	cli.Plan.AlbumMax = &albumMax

	fmt.Fprintln(stdout, "demo: uploading 2 test images to Saved Messages...")

	code := RunUpload(configPath, cli, RunOptions{
		NoProgress: false,
		Stdout:     stdout,
	})
	if code != 0 {
		return fmt.Errorf("upload exited with code %d", code)
	}
	return nil
}

func generateTestImage(path string, c color.Color) error {
	const size = 200
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
