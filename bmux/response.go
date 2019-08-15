package bmux

import "net/http"

// response represents the HTTP response used in the given context.
type response struct {
	inner http.ResponseWriter
}
