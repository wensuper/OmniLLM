package omnillm

import (
	"bytes"
	"errors"
	"testing"
)

// A minimal real PNG header and a minimal JPEG start, enough for type sniffing.
var (
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), make([]byte, 32)...)
	jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}, make([]byte, 32)...)
)

func req(imgs ...Image) Request {
	return Request{Model: "m", Caller: Interactive, Messages: []Message{{Role: "user", Content: "look", Images: imgs}}}
}

func TestImagesAccepted(t *testing.T) {
	r := req(Image{MIME: "image/png", Data: pngBytes}, Image{MIME: "image/jpeg", Data: jpegBytes})
	if err := r.Validate(); err != nil || !r.HasImages() {
		t.Fatal(err)
	}
}

func TestBadImagesRefused(t *testing.T) {
	bad := []Request{
		req(Image{MIME: "image/png", Data: jpegBytes}),            // says PNG, is JPEG
		req(Image{MIME: "image/svg+xml", Data: []byte("<svg/>")}), // not a raster type
		req(Image{MIME: "image/png", Data: []byte("<html>")}),     // not an image at all
		req(Image{MIME: "image/png", Data: append(pngBytes, bytes.Repeat([]byte{0}, MaxImageBytes)...)}),
		req(Image{MIME: "image/png"}),
		req(Image{MIME: "image/png", Data: pngBytes}, Image{MIME: "image/png", Data: pngBytes},
			Image{MIME: "image/png", Data: pngBytes}, Image{MIME: "image/png", Data: pngBytes}, Image{MIME: "image/png", Data: pngBytes}),
		{Model: "m", Caller: Interactive, Messages: []Message{{Role: "assistant", Content: "x", Images: []Image{{MIME: "image/png", Data: pngBytes}}}}},
	}
	for i, r := range bad {
		var e *Error
		if err := r.Validate(); !errors.As(err, &e) || e.Msg != MsgBadImage {
			t.Errorf("case %d: %v", i, err)
		}
	}
}
