//go:build js

package device

// roundTripper stands in for http.RoundTripper in the browser, which picks
// the network path itself; Options.Transport is ignored there. Keeping
// net/http out makes the web client about half a megabyte smaller.
type roundTripper = any
