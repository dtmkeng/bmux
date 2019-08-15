package bmux

import (
	stdContext "context"
	"net/http"
)

// request represents the HTTP request used in the given context.
type request struct {
	inner *http.Request
}

// Context returns the request context.
func (req *request) Context() stdContext.Context {
	return req.inner.Context()
}

// Header returns the header value for the given key.
func (req *request) Header(key string) string {
	return req.inner.Header.Get(key)
}
