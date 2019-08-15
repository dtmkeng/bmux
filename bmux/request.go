package bmux

import (
	stdContext "context"
	"net/http"
)

// Request is an interface for HTTP requests.
type Request interface {
	// Body() Body
	Context() stdContext.Context
	Header(string) string
	// Host() string
	// Internal() *http.Request
	// Method() string
	Path() string
	// Protocol() string
	// Scheme() string
}

// request represents the HTTP request used in the given context.
type request struct {
	inner *http.Request
}

// Path returns the requested path.
func (req *request) Path() string {
	return req.inner.URL.Path
}

// Context returns the request context.
func (req *request) Context() stdContext.Context {
	return req.inner.Context()
}

// Header returns the header value for the given key.
func (req *request) Header(key string) string {
	return req.inner.Header.Get(key)
}
