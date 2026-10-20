//go:build !js

package omnillm

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"

	"golang.org/x/image/webp"
)

func sniffImage(b []byte) string { return http.DetectContentType(b) }

// imageSize reads only the header, not the pixels.
func imageSize(mime string, b []byte) (w, h int, err error) {
	var cfg image.Config
	switch mime {
	case "image/png":
		cfg, err = png.DecodeConfig(bytes.NewReader(b))
	case "image/jpeg":
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(b))
	case "image/gif":
		cfg, err = gif.DecodeConfig(bytes.NewReader(b))
	case "image/webp":
		cfg, err = webp.DecodeConfig(bytes.NewReader(b))
	}
	return cfg.Width, cfg.Height, err
}
