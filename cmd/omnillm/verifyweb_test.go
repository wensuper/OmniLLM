package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A web host and a release, both faked: the page matches the release, then
// one file is changed, then the page is a build no release made.
func TestVerifyWebComparesEveryFileWithTheSignedRelease(t *testing.T) {
	files := map[string]string{
		"index.html": "<html>page</html>",
		"app.js":     "const x = 1;\nconst BUILD = \"0123456789abcdef\";\n",
		"omni.wasm":  "wasm bytes",
	}
	sum := func(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
	man := webManifest{Build: "0123456789abcdef", Files: map[string]string{}}
	for p, b := range files {
		man.Files[p] = sum(b)
	}
	manJSON, _ := json.Marshal(man)
	signed := sum(string(manJSON)) // what the release's checksums.txt lists
	served := map[string]string{}
	for p, b := range files {
		served[p] = b
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases":
			fmt.Fprint(w, `[{"tag_name":"v0.2.0-alpha.99"},{"tag_name":"v0.2.0-alpha.98"}]`)
		case r.URL.Path == "/dl/v0.2.0-alpha.99/"+webManifestName:
			w.Write(manJSON)
		case r.URL.Path == "/dl/v0.2.0-alpha.99/checksums.txt":
			fmt.Fprintf(w, "%s  omnillm-web.tar.gz\n%s  %s\n", sum("tar"), signed, webManifestName)
		case strings.HasPrefix(r.URL.Path, "/dl/"):
			http.NotFound(w, r)
		case strings.HasPrefix(r.URL.Path, "/web/"):
			b, ok := served[strings.TrimPrefix(r.URL.Path, "/web/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, b)
		}
	}))
	defer srv.Close()
	defer func(u string, f func(string, string) string) { releasesURL, releaseFile = u, f }(releasesURL, releaseFile)
	releasesURL = srv.URL + "/releases"
	releaseFile = func(tag, name string) string { return srv.URL + "/dl/" + tag + "/" + name }
	ctx := context.Background()

	w := verifyWeb(ctx, srv.Client(), srv.URL+"/web/")
	if !w.OK || w.Release != "v0.2.0-alpha.99" || w.Files != 3 {
		t.Fatalf("the release's own page: %+v", w)
	}
	served["omni.wasm"] = "changed"
	if w = verifyWeb(ctx, srv.Client(), srv.URL+"/web/"); w.OK || !w.Mismatch() || len(w.Bad) != 1 || w.Bad[0] != "omni.wasm" {
		t.Fatalf("a changed file passed: %+v", w)
	}
	if !strings.Contains(webCheckText(w), "omni.wasm") {
		t.Fatal(webCheckText(w))
	}
	served["omni.wasm"] = files["omni.wasm"]
	served["app.js"] = strings.Replace(files["app.js"], "0123456789abcdef", "fedcba9876543210", 1)
	if w = verifyWeb(ctx, srv.Client(), srv.URL+"/web/"); !w.Mismatch() || w.Bad[0] != "build" {
		t.Fatalf("a build no release made passed: %+v", w)
	}
	// A manifest that isn't the one in the signed checksums is not trusted.
	served["app.js"] = files["app.js"]
	manJSON = append(manJSON, ' ')
	if w = verifyWeb(ctx, srv.Client(), srv.URL+"/web/"); w.OK || w.Error == "" {
		t.Fatalf("an unsigned manifest was trusted: %+v", w)
	}
}

// Releases from before manifests: the signed web tarball is read instead.
func TestVerifyWebReadsOlderReleasesTarball(t *testing.T) {
	files := map[string]string{"app.js": "const BUILD = \"aaaaaaaaaaaaaaaa\";\n", "index.html": "<html>"}
	var tgz bytes.Buffer
	zw := gzip.NewWriter(&tgz)
	tw := tar.NewWriter(zw)
	for p, b := range files {
		_ = tw.WriteHeader(&tar.Header{Name: "./" + p, Mode: 0o644, Size: int64(len(b)), Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte(b))
	}
	tw.Close()
	zw.Close()
	h := sha256.Sum256(tgz.Bytes())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases":
			fmt.Fprint(w, `[{"tag_name":"v0.2.0-alpha.42"}]`)
		case "/dl/v0.2.0-alpha.42/omnillm-web.tar.gz":
			w.Write(tgz.Bytes())
		case "/dl/v0.2.0-alpha.42/checksums.txt":
			fmt.Fprintf(w, "%s  omnillm-web.tar.gz\n", hex.EncodeToString(h[:]))
		default:
			if b, ok := files[strings.TrimPrefix(r.URL.Path, "/web/")]; ok {
				fmt.Fprint(w, b)
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	defer func(u string, f func(string, string) string) { releasesURL, releaseFile = u, f }(releasesURL, releaseFile)
	releasesURL = srv.URL + "/releases"
	releaseFile = func(tag, name string) string { return srv.URL + "/dl/" + tag + "/" + name }
	if w := verifyWeb(context.Background(), srv.Client(), srv.URL+"/web/"); !w.OK || w.Release != "v0.2.0-alpha.42" || w.Files != 2 {
		t.Fatalf("%+v", w)
	}
}
