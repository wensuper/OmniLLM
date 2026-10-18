// Package qrcode draws the pairing code as a QR code for a terminal or the
// console. The QR code carries exactly the pairing text, which is a secret:
// it is only ever shown on the hub's own screen.
package qrcode

import (
	"fmt"
	"strings"

	"rsc.io/qr"
)

const quiet = 2 // modules of white border

// Terminal renders text as a QR code using half-block characters, two
// modules per character row. Dark terminals and light terminals both scan,
// because the code is drawn as explicit dark-on-light.
func Terminal(text string) (string, error) {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}
	n := c.Size
	black := func(x, y int) bool {
		if x < 0 || y < 0 || x >= n || y >= n {
			return false
		}
		return c.Black(x, y)
	}
	var b strings.Builder
	// Light background, dark modules: ANSI reverse would vary by theme, so
	// use explicit colors (black on white).
	const on, off = "\x1b[30;47m", "\x1b[0m"
	for y := -quiet; y < n+quiet; y += 2 {
		b.WriteString(on)
		for x := -quiet; x < n+quiet; x++ {
			top, bottom := black(x, y), black(x, y+1)
			switch {
			case top && bottom:
				b.WriteRune('█')
			case top:
				b.WriteRune('▀')
			case bottom:
				b.WriteRune('▄')
			default:
				b.WriteRune(' ')
			}
		}
		b.WriteString(off)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// SVG renders text as a QR code (dark modules on white, 4-module border).
func SVG(text string) (string, error) {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}
	n := c.Size
	const border = 4
	var path strings.Builder
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if c.Black(x, y) {
				fmt.Fprintf(&path, "M%d %dh1v1h-1z", x+border, y+border)
			}
		}
	}
	size := n + 2*border
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+
		`<rect width="%d" height="%d" fill="#fff"/><path d="%s" fill="#000"/></svg>`, size, size, size, size, path.String()), nil
}
