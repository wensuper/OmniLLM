package main

// Checking the web client against the release. Phones load the page from
// a web host (omni.ixxzh.com, run by whoever runs the public service, or
// the owner's own). Encryption can't protect against that host serving a
// changed page, so it is made checkable instead: every release publishes
// a manifest of the page's files (omnillm-web-manifest.json, in the signed
// checksums.txt), anyone can run `omnillm verify web`, and every hub checks
// the page its own devices use, at start and once a day, and says so when
// it differs. The check needs nothing from the web host's operator.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wensuper/OmniLLM/hub"
)

// webManifest lists the page's files as a release built them.
type webManifest struct {
	Build string            `json:"build"`
	Files map[string]string `json:"files"` // path → sha256 (hex)
}

// webCheck is what the last check found. OK: every file matched a signed
// release. Bad: files that differ or are missing (or "build" when the
// page's build is in no recent release). Error: the check couldn't run
// (network); that says nothing about the page.
type webCheck struct {
	At      time.Time `json:"at"`
	URL     string    `json:"url"`
	Build   string    `json:"build,omitempty"`
	Release string    `json:"release,omitempty"`
	Files   int       `json:"files,omitempty"`
	OK      bool      `json:"ok"`
	Bad     []string  `json:"bad,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// Mismatch: the page differs from every release (not a failed check).
func (w *webCheck) Mismatch() bool { return w != nil && !w.OK && w.Error == "" }

// releaseFile is a file published with a release (a variable for tests).
var releaseFile = func(tag, name string) string {
	return "https://github.com/wensuper/OmniLLM/releases/download/" + tag + "/" + name
}

const webManifestName = "omnillm-web-manifest.json"

var buildStamp = regexp.MustCompile(`(?m)^const BUILD = "([0-9a-f]{16})";`)

// verifyWeb checks the page at base ("https://omni.ixxzh.com/") file by
// file against the release that built it.
func verifyWeb(ctx context.Context, c *http.Client, base string) webCheck {
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	w := webCheck{At: time.Now(), URL: base}
	app, err := get(ctx, c, base+"app.js", 8<<20)
	if err != nil {
		w.Error = err.Error()
		return w
	}
	m := buildStamp.FindSubmatch(app)
	if m == nil {
		w.Bad = []string{"app.js"}
		return w
	}
	w.Build = string(m[1])
	man, tag, err := findManifest(ctx, c, w.Build)
	if err != nil {
		w.Error = err.Error()
		return w
	}
	if man == nil {
		w.Bad = []string{"build"} // no recent release built this page
		return w
	}
	w.Release = tag
	paths := make([]string, 0, len(man.Files))
	for p := range man.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		b, err := get(ctx, c, base+p, 64<<20)
		if err != nil {
			if strings.Contains(err.Error(), "HTTP 404") {
				w.Bad = append(w.Bad, p)
				continue
			}
			w.Error = err.Error()
			return w
		}
		if sum := sha256.Sum256(b); hex.EncodeToString(sum[:]) != man.Files[p] {
			w.Bad = append(w.Bad, p)
		}
	}
	w.Files = len(paths)
	w.OK = len(w.Bad) == 0
	return w
}

// findManifest looks through recent releases for the one whose web client
// has this build, and checks its manifest against that release's
// checksums.txt (which the release workflow signs). nil: none has it.
func findManifest(ctx context.Context, c *http.Client, build string) (*webManifest, string, error) {
	b, err := get(ctx, c, releasesURL, 4<<20)
	if err != nil {
		return nil, "", err
	}
	var rels []release
	if err := json.Unmarshal(b, &rels); err != nil {
		return nil, "", err
	}
	tried := 0
	for _, r := range rels {
		if r.Draft || tried >= 15 {
			continue
		}
		tried++
		mb, err := get(ctx, c, releaseFile(r.Tag, webManifestName), 1<<20)
		name := webManifestName
		var man *webManifest
		switch {
		case err == nil:
			var m webManifest
			if json.Unmarshal(mb, &m) != nil || m.Build != build {
				continue
			}
			man = &m
		case strings.Contains(err.Error(), "HTTP 404"):
			// Released before manifests (alpha.42 and older): the signed
			// web tarball itself says what each file was.
			name = "omnillm-web.tar.gz"
			if mb, err = get(ctx, c, releaseFile(r.Tag, name), 64<<20); err != nil {
				if strings.Contains(err.Error(), "HTTP 404") {
					continue
				}
				return nil, "", err
			}
			if man, err = manifestOf(mb); err != nil || man.Build != build {
				continue
			}
		default:
			return nil, "", err
		}
		sums, err := get(ctx, c, releaseFile(r.Tag, "checksums.txt"), 1<<20)
		if err != nil {
			return nil, "", err
		}
		sum := sha256.Sum256(mb)
		if !listed(sums, hex.EncodeToString(sum[:]), name) {
			return nil, "", fmt.Errorf("%s: %s is not the one in checksums.txt", r.Tag, name)
		}
		return man, r.Tag, nil
	}
	return nil, "", nil
}

// manifestOf reads a web tarball (omnillm-web.tar.gz) into a manifest.
func manifestOf(tgz []byte) (*webManifest, error) {
	zr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(zr)
	m := &webManifest{Files: map[string]string{}}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, 64<<20))
		if err != nil {
			return nil, err
		}
		p := strings.TrimPrefix(h.Name, "./")
		sum := sha256.Sum256(b)
		m.Files[p] = hex.EncodeToString(sum[:])
		if p == "app.js" {
			if s := buildStamp.FindSubmatch(b); s != nil {
				m.Build = string(s[1])
			}
		}
	}
	return m, nil
}

// listed: checksums.txt has this sum for this name.
func listed(sums []byte, sum, name string) bool {
	for _, l := range bytes.Split(sums, []byte("\n")) {
		f := strings.Fields(string(l))
		if len(f) == 2 && f[0] == sum && strings.TrimPrefix(f[1], "*") == name {
			return true
		}
	}
	return false
}

// webCheckText says a check's result for a person.
func webCheckText(w webCheck) string {
	switch {
	case w.OK:
		return fmt.Sprintf(t("手机用的网页（%s）和 GitHub 上签名发布的 %s 完全一致（构建 %s，%d 个文件逐个核对）。",
			"The web page your phones use (%s) is exactly what the signed GitHub release %s built (build %s, %d files checked one by one)."),
			w.URL, w.Release, w.Build, w.Files)
	case w.Error != "":
		return fmt.Sprintf(t("没能核对手机用的网页（%s）：%s。稍后再试。", "Couldn’t check the web page your phones use (%s): %s. Try again later."), w.URL, w.Error)
	case len(w.Bad) == 1 && w.Bad[0] == "build":
		return fmt.Sprintf(t("警告：手机用的网页（%s，构建 %s）不是最近任何一个签名发布构建的。在弄清楚之前，不要在手机上用它。",
			"Warning: the web page your phones use (%s, build %s) wasn’t built by any recent signed release. Don’t use it on your phones until this is cleared up."), w.URL, w.Build)
	default:
		return fmt.Sprintf(t("警告：手机用的网页（%s）和签名发布的 %s 不一致：%s。在弄清楚之前，不要在手机上用它。",
			"Warning: the web page your phones use (%s) differs from the signed release %s: %s. Don’t use it on your phones until this is cleared up."),
			w.URL, w.Release, strings.Join(w.Bad, t("、", ", ")))
	}
}

// verifyCmd is "omnillm verify web [URL]".
func verifyCmd(args []string) {
	if len(args) == 0 || args[0] != "web" || len(args) > 2 {
		dieArgs(usageErr(t("用法：omnillm verify web [网页地址]", "Usage: omnillm verify web [web address]")))
	}
	base := webAppFromEnv()
	if len(args) == 2 {
		base = args[1]
	}
	if base == "" {
		base = hub.DefaultWebAppURL
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	w := verifyWeb(ctx, &http.Client{Transport: updateClient().Transport, Timeout: 2 * time.Minute}, base)
	fmt.Println(webCheckText(w))
	switch {
	case w.OK:
	case w.Error != "":
		os.Exit(2)
	default:
		os.Exit(1)
	}
}

// The hub's own check: kept in memory for status and in a file across
// restarts.
var (
	webMu   sync.Mutex
	webLast *webCheck
)

func webCheckPath(dir string) string { return filepath.Join(dir, "webcheck.json") }

func lastWebCheck(dir string) *webCheck {
	webMu.Lock()
	defer webMu.Unlock()
	if webLast == nil {
		var w webCheck
		if b, err := os.ReadFile(webCheckPath(dir)); err == nil && json.Unmarshal(b, &w) == nil {
			webLast = &w
		}
	}
	return webLast
}

func saveWebCheck(dir string, w webCheck) {
	webMu.Lock()
	defer webMu.Unlock()
	webLast = &w
	if b, err := json.MarshalIndent(w, "", "  "); err == nil {
		_ = os.WriteFile(webCheckPath(dir), b, 0o600)
	}
}

// watchWeb checks the page this hub's devices open, a few minutes after
// start and then daily; a failed check (network) is tried again in an
// hour and leaves the last real result in place.
func watchWeb(ctx context.Context, dir, base string) {
	if base == "" {
		base = hub.DefaultWebAppURL
	}
	wait := 3 * time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		w := verifyWeb(cctx, &http.Client{Transport: updateClient().Transport, Timeout: 2 * time.Minute}, base)
		cancel()
		if ctx.Err() != nil {
			return
		}
		wait = 24 * time.Hour
		if w.Error != "" {
			wait = time.Hour
			if prev := lastWebCheck(dir); prev != nil && prev.URL == w.URL && prev.Error == "" {
				continue
			}
		}
		saveWebCheck(dir, w)
		if w.Mismatch() {
			fmt.Fprintln(os.Stderr, "omnillm: "+webCheckText(w))
		}
	}
}
