package qrcode

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	gzqr "github.com/makiuchi-d/gozxing/qrcode"
	"rsc.io/qr"
)

// decode reads the QR code back, proving it scans to exactly the text.
func decode(t *testing.T, text string) string {
	t.Helper()
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		t.Fatal(err)
	}
	const scale, border = 4, 4
	n := (c.Size + 2*border) * scale
	img := image.NewGray(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			mx, my := x/scale-border, y/scale-border
			v := color.Gray{255}
			if mx >= 0 && my >= 0 && mx < c.Size && my < c.Size && c.Black(mx, my) {
				v = color.Gray{0}
			}
			img.SetGray(x, y, v)
		}
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatal(err)
	}
	res, err := gzqr.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res.GetText()
}

func TestPairingCodeScansBack(t *testing.T) {
	// A realistic pairing text (≈ 350 characters).
	text := "omnillm1-" + strings.Repeat("eyJyIjoid3NzOi8vcmVsYXkuZXhhbXBsZS5jb20iLCJtIjoi", 7)
	if got := decode(t, text); got != text {
		t.Fatalf("scanned %q", got)
	}
}

func TestRenderers(t *testing.T) {
	s, err := Terminal("omnillm1-abc")
	if err != nil || !strings.Contains(s, "█") || strings.Count(s, "\n") < 10 {
		t.Fatalf("terminal render: %v %q", err, s)
	}
	svg, err := SVG("omnillm1-abc")
	if err != nil || !strings.HasPrefix(svg, "<svg") || strings.Contains(svg, "<script") || !strings.Contains(svg, "fill=\"#000\"") {
		t.Fatalf("svg: %v", err)
	}
	if _, err := SVG(strings.Repeat("x", 5000)); err == nil {
		t.Fatal("over-long text should not encode")
	}
}
