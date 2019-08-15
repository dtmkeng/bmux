package bmux

import "net/http"

// request represents the HTTP request used in the given context.
type request struct {
	inner *http.Request
}
