package goodgopher

import "net/http"

// New returns an http.Handler that provides the entrypoints needed for goodgopher.
func New(appID int, key, scret []byte, transport http.RoundTripper) (http.Handler, error) {
	return nil, nil
}
