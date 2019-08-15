package bmux

import "net/http"

// response represents the HTTP response used in the given context.
type response struct {
	inner http.ResponseWriter
}

// Response is the interface for an HTTP response.
type Response interface {
	// Header(string) string
	Internal() http.ResponseWriter
	// SetHeader(string, string)
	// SetInternal(http.ResponseWriter)
}

// Internal returns the underlying http.ResponseWriter.
// This method should be avoided unless absolutely necessary
// because Aero doesn't guarantee that the underlying framework
// will always stay net/http based in the future.
func (res *response) Internal() http.ResponseWriter {
	return res.inner
}
