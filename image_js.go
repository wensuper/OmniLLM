//go:build js

package omnillm

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// sniffImage matches what http.DetectContentType says for the four image
// types; net/http is too large for the browser build.
func sniffImage(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case bytes.HasPrefix(b, []byte("\xff\xd8\xff")):
		return "image/jpeg"
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return "image/gif"
	case len(b) >= 14 && string(b[:4]) == "RIFF" && string(b[8:14]) == "WEBPVP":
		return "image/webp"
	}
	return ""
}

// imageSize reads PNG and GIF sizes straight from their fixed headers.
// JPEG and WebP are not checked in the browser: their decoders cost the web
// client about 70 KB, and the hub decodes every image's header before using
// it.
func imageSize(mime string, b []byte) (w, h int, err error) {
	switch mime {
	case "image/png": // signature, IHDR length and type, then width, height
		if len(b) < 24 || string(b[12:16]) != "IHDR" {
			return 0, 0, errBadHeader
		}
		return int(binary.BigEndian.Uint32(b[16:])), int(binary.BigEndian.Uint32(b[20:])), nil
	case "image/gif": // signature, then the logical screen size
		if len(b) < 10 {
			return 0, 0, errBadHeader
		}
		return int(binary.LittleEndian.Uint16(b[6:])), int(binary.LittleEndian.Uint16(b[8:])), nil
	}
	return 1, 1, nil
}

var errBadHeader = errors.New("omnillm: bad image header")
