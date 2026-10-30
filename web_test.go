package omnillm

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The web client holds the device's keys, so what a model writes is never
// parsed as HTML: replies are built from createElement and text nodes, and
// only the inline tags in HTML_TAGS become elements, without attributes.
func TestWebClientNeverParsesHTML(t *testing.T) {
	b, err := os.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "DOMParser", "createContextualFragment", "setAttribute(\"style", "eval(", "new Function"} {
		if strings.Contains(js, bad) {
			t.Errorf("web/app.js uses %s", bad)
		}
	}
	m := regexp.MustCompile(`const HTML_TAGS = new Set\(\[([^\]]*)\]\)`).FindStringSubmatch(js)
	if m == nil {
		t.Fatal("HTML_TAGS not found in web/app.js")
	}
	allowed := map[string]bool{"b": true, "strong": true, "i": true, "em": true, "u": true, "s": true, "del": true, "ins": true, "mark": true, "sub": true, "sup": true, "small": true, "kbd": true, "code": true, "br": true}
	for _, tag := range regexp.MustCompile(`"([a-z0-9]+)"`).FindAllStringSubmatch(m[1], -1) {
		if !allowed[tag[1]] {
			t.Errorf("HTML_TAGS lets a model make <%s>; only inline text tags belong there", tag[1])
		}
	}
	// Links go only to http(s) and mailto.
	if !strings.Contains(js, `\]\(((?:https?:\/\/|mailto:)`) {
		t.Error("Markdown links in web/app.js are not limited to http(s) and mailto")
	}
	// Math: Temml without trust, so TeX adds no \href, \class or \style.
	if strings.Count(js, "temml.render(") != 1 || !strings.Contains(js, "trust: false }") {
		t.Error("web/app.js must call temml.render once, with trust: false")
	}
}

// Vendored scripts run with the page's rights: each is pinned by its
// sha256, which scripts/build-web.sh checks before every build.
func TestWebVendorPinned(t *testing.T) {
	for _, sum := range []string{"web/vendor/jsqr.sha256", "web/vendor/temml.sha256"} {
		if _, err := os.Stat(sum); err != nil {
			t.Errorf("%s missing", sum)
		}
	}
	sw, err := os.ReadFile("web/sw.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"theme.js", "vendor/jsqr.js", "vendor/temml.min.js", "vendor/temml.css"} {
		if !strings.Contains(string(sw), `"`+f+`"`) {
			t.Errorf("web/sw.js doesn't keep %s for offline use", f)
		}
	}
}
