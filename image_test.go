package omnillm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encoded(t *testing.T, w, h int, kind string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{200, 20, 20, 255})
	var b bytes.Buffer
	var err error
	if kind == "png" {
		err = png.Encode(&b, img)
	} else {
		err = jpeg.Encode(&b, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// hugePNG is a tiny file whose header claims w x h pixels.
func hugePNG(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 2
	chunk := func(typ string, data []byte) []byte {
		var c bytes.Buffer
		binary.Write(&c, binary.BigEndian, uint32(len(data)))
		c.WriteString(typ)
		c.Write(data)
		binary.Write(&c, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(typ), data...)))
		return c.Bytes()
	}
	out := []byte("\x89PNG\r\n\x1a\n")
	out = append(out, chunk("IHDR", ihdr)...)
	return append(out, chunk("IEND", nil)...)
}

func req(imgs ...Image) Request {
	return Request{Model: "m", Caller: Interactive, Messages: []Message{{Role: "user", Content: "look", Images: imgs}}}
}

func TestImagesAccepted(t *testing.T) {
	r := req(Image{MIME: "image/png", Data: encoded(t, 64, 64, "png")}, Image{MIME: "image/jpeg", Data: encoded(t, 1600, 1200, "jpeg")})
	if err := r.Validate(); err != nil || !r.HasImages() {
		t.Fatal(err)
	}
}

func TestBadImagesRefused(t *testing.T) {
	p, j := encoded(t, 8, 8, "png"), encoded(t, 8, 8, "jpeg")
	bad := []Request{
		req(Image{MIME: "image/png", Data: j}),                    // says PNG, is JPEG
		req(Image{MIME: "image/svg+xml", Data: []byte("<svg/>")}), // not a raster type
		req(Image{MIME: "image/png", Data: []byte("<html>")}),     // not an image at all
		req(Image{MIME: "image/png", Data: append(append([]byte{}, p...), bytes.Repeat([]byte{0}, MaxImageBytes)...)}),
		req(Image{MIME: "image/png"}),
		req(Image{MIME: "image/png", Data: hugePNG(60000, 60000)}), // a few bytes claiming 3.6 gigapixels
		req(Image{MIME: "image/png", Data: hugePNG(9000, 10)}),     // one side too long
		req(Image{MIME: "image/png", Data: p}, Image{MIME: "image/png", Data: p}, Image{MIME: "image/png", Data: p},
			Image{MIME: "image/png", Data: p}, Image{MIME: "image/png", Data: p}),
		{Model: "m", Caller: Interactive, Messages: []Message{{Role: "assistant", Content: "x", Images: []Image{{MIME: "image/png", Data: p}}}}},
	}
	for i, r := range bad {
		var e *Error
		if err := r.Validate(); !errors.As(err, &e) || e.Msg != MsgBadImage {
			t.Errorf("case %d: %v", i, err)
		}
	}
}
