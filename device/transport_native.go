//go:build !js

package device

import "net/http"

// roundTripper is http.RoundTripper; the browser build has no net/http.
type roundTripper = http.RoundTripper
