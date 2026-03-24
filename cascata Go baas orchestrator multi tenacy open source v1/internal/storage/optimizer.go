package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"

	"github.com/disintegration/imaging"
)

// MediaOptimizer provides on-the-fly image transformations (Phase 18).
// It acts as the "CDN Edge Optimizer" within the Cascata orchestrator.
type MediaOptimizer struct{}

func NewMediaOptimizer() *MediaOptimizer {
	return &MediaOptimizer{}
}

// Transform processes a source image stream and applies resizing/conversion.
func (o *MediaOptimizer) Transform(r io.Reader, width, height int, format string) ([]byte, string, error) {
	slog.Debug("storage.optimizer: starting transformation", "w", width, "h", height, "fmt", format)

	// 1. Decode Source
	src, fname, err := image.Decode(r)
	if err != nil {
		return nil, "", fmt.Errorf("optimizer.decode: %w", err)
	}

	// 2. Resize
	// If one dimension is 0, imaging will preserve aspect ratio.
	dst := imaging.Resize(src, width, height, imaging.Lanczos)

	// 3. Encode to Target Format
	buf := new(bytes.Buffer)
	var mime string

	targetFmt := format
	if targetFmt == "" {
		targetFmt = fname
	}

	switch targetFmt {
	case "jpeg", "jpg":
		err = jpeg.Encode(buf, dst, &jpeg.Options{Quality: 85})
		mime = "image/jpeg"
	case "png":
		err = png.Encode(buf, dst)
		mime = "image/png"
	case "gif":
		err = gif.Encode(buf, dst, nil)
		mime = "image/gif"
	default:
		return nil, "", fmt.Errorf("unsupported output format: %s", targetFmt)
	}

	if err != nil {
		return nil, "", fmt.Errorf("optimizer.encode: %w", err)
	}

	return buf.Bytes(), mime, nil
}
