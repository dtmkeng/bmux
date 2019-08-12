package bmux

import (
	"fmt"
	"net/http"
)

type matcher interface {
	Match(*http.Request, *RouteMatch) bool
}

// methodMatcher matches the request against HTTP methods.
type methodMatcher []string

// Route stores information to match a request and build URLs.
type Route struct {
	// Request handler for the route.
	handler http.Handler
	// If true, this route never matches: it is only used to build URLs.
	buildOnly bool
	// The name used to build URLs.
	name string
	// Error resulted from building a route.
	err error

	// "global" reference to all named routes
	// namedRoutes map[string]*Route

	// config possibly passed in from `Router`
	routeConf
}

// SkipClean reports whether path cleaning is enabled for this route via
// Router.SkipClean.
func (r *Route) SkipClean() bool {
	return r.skipClean
}

// Match matches the route against the request.
func (r *Route) Match(req *http.Request, match *RouteMatch) bool {
	if r.buildOnly || r.err != nil {
		return false
	}

	var matchErr error

	// Match everything.
	for _, m := range r.matchers {
		if matched := m.Match(req, match); !matched {
			if _, ok := m.(methodMatcher); ok {
				matchErr = ErrMethodMismatch
				continue
			}

			// Ignore ErrNotFound errors. These errors arise from match call
			// to Subrouters.
			//
			// This prevents subsequent matching subrouters from failing to
			// run middleware. If not ignored, the middleware would see a
			// non-nil MatchErr and be skipped, even when there was a
			// matching route.
			if match.MatchErr == ErrNotFound {
				match.MatchErr = nil
			}

			matchErr = nil
			return false
		}
	}

	if matchErr != nil {
		match.MatchErr = matchErr
		return false
	}

	if match.MatchErr == ErrMethodMismatch {
		// We found a route which matches request method, clear MatchErr
		match.MatchErr = nil
		// Then override the mis-matched handler
		match.Handler = r.handler
	}

	// Yay, we have a match. Let's collect some info about it.
	if match.Route == nil {
		match.Route = r
	}
	if match.Handler == nil {
		match.Handler = r.handler
	}
	if match.Vars == nil {
		match.Vars = make(map[string]string)
	}

	// Set variables.
	r.regexp.setMatch(req, match, r)
	return true
}

func (m methodMatcher) Match(r *http.Request, match *RouteMatch) bool {
	return matchInArray(m, r.Method)
}

// matchInArray returns true if the given string value is in the array.
func matchInArray(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

// Lookup finds the handler and parameters for the given route
// and assigns them to the given context.
func (router *Router) Lookup(method string, path string, ctx *contexts) {
	tree := router.selectTree(method)

	// Fast path for the root node
	if tree.prefix == path {
		ctx.handler = tree.data
		return
	}

	tree.find(path, ctx)
}

// Add registers a new handler for the given method and path.
func (router *Router) Add(method string, path string, handler Handler) {
	tree := router.selectTree(method)

	if tree == nil {
		panic(fmt.Errorf("Unknown HTTP method: '%s'", method))
	}

	tree.add(path, handler)
}

// selectTree returns the tree by the given HTTP method.
func (router *Router) selectTree(method string) *tree {
	switch method {
	case http.MethodGet:
		return &router.get
	case http.MethodPost:
		return &router.post
	case http.MethodDelete:
		return &router.delete
	case http.MethodPut:
		return &router.put
	case http.MethodPatch:
		return &router.patch
	case http.MethodHead:
		return &router.head
	case http.MethodConnect:
		return &router.connect
	case http.MethodTrace:
		return &router.trace
	case http.MethodOptions:
		return &router.options
	default:
		return nil
	}
}
