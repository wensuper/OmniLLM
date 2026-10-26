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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	// Signed: checksums.txt's signature was checked too (cosign is
	// installed); otherwise the checksums come from GitHub over TLS.
	Signed bool     `json:"signed,omitempty"`
	OK     bool     `json:"ok"`
	Bad    []string `json:"bad,omitempty"`
	Error  string   `json:"error,omitempty"`
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
	// As copied from a phone's address bar (or set as OMNILLM_WEB_APP):
	// the page's folder, without a pairing link (#pair=…), a query or
	// index.html.
	if u, err := url.Parse(base); err == nil {
		u.RawQuery, u.Fragment, u.RawFragment = "", "", ""
		if strings.HasSuffix(u.Path, ".html") {
			u.Path = u.Path[:strings.LastIndex(u.Path, "/")+1]
		}
		base = u.String()
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	w := webCheck{At: time.Now(), URL: base}
	app, err := get(ctx, c, base+"app.js", 8<<20)
	if err != nil {
		w.Error = "web" // the page itself didn't answer
		if strings.Contains(err.Error(), "HTTP 404") {
			w.Error = "web_none" // it answered, with no OmniLLM page there
		}
		return w
	}
	m := buildStamp.FindSubmatch(app)
	if m == nil {
		w.Bad = []string{"app.js"}
		return w
	}
	w.Build = string(m[1])
	man, tag, sums, err := findManifest(ctx, c, w.Build)
	if errors.Is(err, errNotListed) {
		w.Release, w.Bad = tag, []string{"release"} // the release's own checksums disown its file list
		return w
	}
	if err != nil {
		w.Error = githubError(err)
		return w
	}
	if man == nil {
		w.Bad = []string{"build"} // no recent release built this page
		return w
	}
	w.Release = tag
	signed, err := signedRelease(ctx, c, tag, sums)
	if errors.Is(err, errBadSignature) {
		w.Bad = []string{"signature"}
		return w
	}
	if err != nil { // couldn't check (a download, cosign itself): not a verdict
		w.Error = githubError(err)
		return w
	}
	w.Signed = signed
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
			w.Error = "web"
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
// checksums.txt (which the release workflow signs), returned too so that
// the signature is checked on these very bytes. nil: none has it.
func findManifest(ctx context.Context, c *http.Client, build string) (*webManifest, string, []byte, error) {
	rels, err := listReleases(ctx, c)
	if err != nil {
		return nil, "", nil, err
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
				return nil, "", nil, err
			}
			if man, err = manifestOf(mb); err != nil || man.Build != build {
				continue
			}
		default:
			return nil, "", nil, err
		}
		sums, err := get(ctx, c, releaseFile(r.Tag, "checksums.txt"), 1<<20)
		if err != nil {
			return nil, "", nil, err
		}
		sum := sha256.Sum256(mb)
		if !listed(sums, hex.EncodeToString(sum[:]), name) {
			return nil, r.Tag, nil, fmt.Errorf("%s: %s: %w", r.Tag, name, errNotListed)
		}
		return man, r.Tag, sums, nil
	}
	return nil, "", nil, nil
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

// lookCosign finds cosign (a variable for tests: they must not depend on
// whether this computer has it).
var lookCosign = func() (string, error) { return exec.LookPath("cosign") }

// signedRelease checks a release's checksums.txt against its signature
// with cosign, pinned to this repository's release workflow (as install.sh
// does): the sigstore bundle, or the older certificate and signature.
// false, nil: cosign isn't installed, so only TLS and the checksums vouch
// for it; errBadSignature: wrong, or missing (every release publishes one,
// so a missing one is not "unsigned"); other errors: couldn't check.
func signedRelease(ctx context.Context, c *http.Client, tag string, sums []byte) (bool, error) {
	cosign, err := lookCosign()
	if err != nil {
		return false, nil
	}
	dir, err := os.MkdirTemp("", "omnillm-verify-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(dir)
	fetch := func(name string) (string, error) {
		b, err := get(ctx, c, releaseFile(tag, name), 1<<20)
		if err != nil {
			if strings.Contains(err.Error(), "HTTP 404") {
				return "", errBadSignature
			}
			return "", err
		}
		p := filepath.Join(dir, name)
		if os.WriteFile(p, b, 0o600) != nil {
			return "", errors.New("verify: can't write a temporary file")
		}
		return p, nil
	}
	var sig []string
	if bp, err := fetch("checksums.txt.sigstore.json"); err == nil {
		sig = []string{"--bundle", bp}
	} else if !errors.Is(err, errBadSignature) {
		return false, err
	} else {
		pem, err := fetch("checksums.txt.pem")
		if err != nil {
			return false, err
		}
		sg, err := fetch("checksums.txt.sig")
		if err != nil {
			return false, err
		}
		sig = []string{"--certificate", pem, "--signature", sg}
	}
	sp := filepath.Join(dir, "checksums.txt")
	if os.WriteFile(sp, sums, 0o600) != nil {
		return false, errors.New("verify: can't write a temporary file")
	}
	args := append(append([]string{"verify-blob"}, sig...),
		"--certificate-identity-regexp", `^https://github\.com/wensuper/OmniLLM/\.github/workflows/release\.yml@refs/tags/v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`,
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com", sp)
	cmd := exec.CommandContext(ctx, cosign, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && !cosignOffline(stderr.String()) {
			return false, errBadSignature // cosign ran and said no
		}
		return false, err // it couldn't get Sigstore's keys: not a verdict
	}
	return true, nil
}

var (
	errBadSignature = errors.New("verify: the signature doesn't check out")
	errNotListed    = errors.New("not the one in checksums.txt")
)

// webCheckText says a check's result for a person. "Signed" only when the
// signature was checked; otherwise "published on GitHub".
func webCheckText(w webCheck) string {
	switch {
	case w.OK && w.Signed:
		return fmt.Sprintf(t("网页版（%s）和 GitHub 上签名发布的 %s 完全一致（构建 %s，%d 个文件逐个核对，签名也核对过）。",
			"The web app (%s) is exactly what the signed GitHub release %s built (build %s, %d files checked one by one, and the signature too)."),
			w.URL, w.Release, w.Build, w.Files)
	case w.OK:
		return fmt.Sprintf(t("网页版（%s）和 GitHub 上发布的 %s 完全一致（构建 %s，%d 个文件逐个核对；装了 cosign 还会核对签名）。",
			"The web app (%s) is exactly what the GitHub release %s built (build %s, %d files checked one by one; with cosign installed, the signature is checked too)."),
			w.URL, w.Release, w.Build, w.Files)
	case len(w.Bad) == 1 && w.Bad[0] == "signature":
		return fmt.Sprintf(t("警告：GitHub 上 %s 的签名核对没有通过。在弄清楚之前，不要在手机上用这个网页（%s）。",
			"Warning: the signature of the GitHub release %s didn’t check out. Don’t use the web page (%s) on your phones until this is cleared up."), w.Release, w.URL)
	case len(w.Bad) == 1 && w.Bad[0] == "release":
		return fmt.Sprintf(t("警告：GitHub 上 %s 的文件清单和它自己的校验和对不上。在弄清楚之前，不要在手机上用这个网页（%s）。",
			"Warning: the GitHub release %s’s list of files doesn’t match its own checksums. Don’t use the web page (%s) on your phones until this is cleared up."), w.Release, w.URL)
	case len(w.Bad) == 1 && w.Bad[0] == "app.js" && w.Release == "":
		return fmt.Sprintf(t("警告：网页版（%s）不是 OmniLLM 发布构建的（找不到构建号）。在弄清楚之前，先别用它。",
			"Warning: the web app (%s) isn’t an OmniLLM release build (it has no build number). Don’t use it until this is cleared up."), w.URL)
	case w.Error == "github_limited":
		return fmt.Sprintf(t("没能核对网页版（%s）：GitHub 暂时不让这台电脑再查了（同一个网络地址每小时最多 60 次）。过一会儿再试。", "Couldn’t check the web app (%s): GitHub isn’t answering this computer for now (it allows 60 checks an hour per network address). Try again in a while."), w.URL)
	case w.Error == "web_none":
		return fmt.Sprintf(t("这个地址上没有 OmniLLM 的网页（%s）。检查地址后再试。", "There’s no OmniLLM web page at %s. Check the address and try again."), w.URL)
	case w.Error == "web":
		return fmt.Sprintf(t("没能核对网页版（%s）：这个网页没有应答。检查网络后再试。", "Couldn’t check the web app (%s): the page didn’t answer. Check the network and try again."), w.URL)
	case w.Error != "":
		return fmt.Sprintf(t("没能核对网页版（%s）：GitHub 没有应答。检查网络后再试。", "Couldn’t check the web app (%s): GitHub didn’t answer. Check the network and try again."), w.URL)
	case len(w.Bad) == 1 && w.Bad[0] == "build":
		return fmt.Sprintf(t("警告：手机用的网页（%s，构建 %s）不是 GitHub 上最近任何一个发布构建的。在弄清楚之前，先别用它。",
			"Warning: the web page your phones use (%s, build %s) wasn’t built by any recent GitHub release. Don’t use it until this is cleared up."), w.URL, w.Build)
	default:
		return fmt.Sprintf(t("警告：网页版（%s）和 GitHub 上发布的 %s 不一致：%s。在弄清楚之前，先别用它。",
			"Warning: the web app (%s) differs from the GitHub release %s: %s. Don’t use it until this is cleared up."),
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
	if !strings.HasPrefix(base, "https://") && !strings.HasPrefix(base, "http://") {
		dieArgs(usageErr(t("网页地址要以 https:// 开头，例如 omnillm verify web https://omni.ixxzh.com", "The web address starts with https://, e.g. omnillm verify web https://omni.ixxzh.com")))
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

// githubError is what a failed GitHub check keeps (said in webCheckText,
// in the language of whoever reads it).
func githubError(err error) string {
	if errors.Is(err, errRateLimited) {
		return "github_limited"
	}
	return "github"
}

// cosignOffline: cosign stopped before verifying because it couldn't reach
// Sigstore (its trust root or transparency log keys), not because the
// signature is wrong. It exits 1 either way. Only the start of its first
// error line is read, cosign's own words: further on it quotes the
// certificate, which whoever signed it chose (a subject naming "tuf").
func cosignOffline(stderr string) bool {
	for _, line := range strings.Split(stderr, "\n") {
		if !strings.HasPrefix(line, "Error: ") {
			continue
		}
		for _, p := range []string{"Error: setting up clients", "Error: getting rekor public keys", "Error: fetching trusted root", "Error: getting trusted root", "Error: tuf:"} {
			if strings.HasPrefix(line, p) {
				return true
			}
		}
		return false
	}
	return false
}
