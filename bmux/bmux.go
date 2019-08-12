package bmux

import (
	"errors"
	"fmt"
	"net/http"
	"path"
)

var (
	// ErrMethodMismatch is returned when the method in the request does not match
	// the method defined against the route.
	ErrMethodMismatch = errors.New("method is not allowed")
	// ErrNotFound is returned when no route match is found.
	ErrNotFound = errors.New("no matching route was found")
)

// RouteMatch stores information about a matched route.
type RouteMatch struct {
	Route   *Route
	Handler http.Handler
	// *Application
	Vars map[string]string
	// aero.Application
	// MatchErr is set to appropriate matching error
	// It is set to ErrMethodMismatch if there is a mismatch in
	// the request method and route method
	MatchErr error
}

// NewRouter returns a new router instance.
func NewRouter() *Router {
	return &Router{namedRoutes: make(map[string]*Route)}
}

// Router ...
type Router struct {
	// Configurable Handler to be used when no route matches.
	NotFoundHandler http.Handler
	// Configurable Handler to be used when the request method does not match the route.
	MethodNotAllowedHandler http.Handler
	*Application
	// Routes to be matched, in order.
	routes []*Route

	// Routes by name for URL building.
	namedRoutes map[string]*Route

	// If true, do not clear the request context after handling the request.
	//
	// Deprecated: No effect when go1.7+ is used, since the context is stored
	// on the request itself.
	KeepContext bool

	// Slice of middlewares to be called after a match is found
	middlewares []middleware

	// configuration shared with `Route`
	routeConf

	get     tree
	post    tree
	delete  tree
	put     tree
	patch   tree
	head    tree
	connect tree
	trace   tree
	options tree
}
type routeConf struct {
	// If true, "/path/foo%2Fbar/to" will match the path "/path/{var}/to"
	useEncodedPath bool

	// If true, when the path pattern is "/path/", accessing "/path" will
	// redirect to the former and vice versa.
	strictSlash bool

	// If true, when the path pattern is "/path//to", accessing "/path//to"
	// will not redirect
	skipClean bool

	// Manager for the variables from host and path.
	regexp routeRegexpGroup

	// List of matchers.
	matchers []matcher

	// The scheme used when building URLs.
	buildScheme string

	// buildVarsFunc BuildVarsFunc
}
type contextKey int

const (
	varsKey contextKey = iota
	routeKey
)

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// contexts.handler = tree.data
	if !r.skipClean {
		path := req.URL.Path
		if r.useEncodedPath {
			path = req.URL.EscapedPath()
		}
		// Clean path to canonical form and redirect.
		if p := cleanPath(path); p != path {

			// Added 3 lines (Philip Schlump) - It was dropping the query string and #whatever from query.
			// This matches with fix in go 1.2 r.c. 4 for same problem.  Go Issue:
			// http://code.google.com/p/go/issues/detail?id=5252
			url := *req.URL
			url.Path = p
			p = url.String()

			w.Header().Set("Location", p)
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
	}
	var match RouteMatch
	var handler http.Handler
	fmt.Println(handler)
	if r.Match(req, &match) {
		handler = match.Handler
		req = setVars(req, match.Vars)
		req = setCurrentRoute(req, match.Route)
	}

	if handler == nil && match.MatchErr == ErrMethodMismatch {
		handler = methodNotAllowedHandler()
	}

	if handler == nil {
		handler = http.NotFoundHandler()
	}

	handler.ServeHTTP(w, req)
}

// ServeHTTP responds to the given request.
type RewriteContext interface {
	Path() string
	SetPath(string)
}

// func (app *Application) ServeHTTP(response http.ResponseWriter, request *http.Request) {
// 	ctx := app.NewContext(request, response)

// 	for _, rewrite := range app.rewrite {
// 		rewrite(ctx)
// 	}

// 	app.router.Lookup(request.Method, request.URL.Path, ctx)

// 	if ctx.handler == nil {
// 		response.WriteHeader(http.StatusNotFound)
// 		ctx.Close()
// 		return
// 	}

// 	err := ctx.handler(ctx)

// 	if err != nil {
// 		for _, callback := range app.onError {
// 			callback(ctx, err)
// 		}
// 	}

// 	ctx.Close()
// }
func setVars(r *http.Request, val interface{}) *http.Request {
	return contextSet(r, varsKey, val)
}

func setCurrentRoute(r *http.Request, val interface{}) *http.Request {
	return contextSet(r, routeKey, val)
}

// SkipClean defines the path cleaning behaviour for new routes. The initial
// value is false. Users should be careful about which routes are not cleaned
//
// When true, if the route path is "/path//to", it will remain with the double
// slash. This is helpful if you have a route like: /fetch/http://xkcd.com/534/
//
// When false, the path will be cleaned, so /fetch/http://xkcd.com/534/ will
func (r *Router) SkipClean(value bool) *Router {
	r.skipClean = value
	return r
}

// cleanPath returns the canonical path for p, eliminating . and .. elements.
// Borrowed from the net/http package.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}

	return np
}

// Match attempts to match the given request against the router's registered routes.
//
// If the request matches a route of this router or one of its subrouters the Route,
// Handler, and Vars fields of the the match argument are filled and this function
// returns true.
//
// If the request does not match any of this router's or its subrouters' routes
// then this function returns false. If available, a reason for the match failure
// will be filled in the match argument's MatchErr field. If the match failure type
// (eg: not found) has a registered handler, the handler is assigned to the Handler
// field of the match argument.
func (r *Router) Match(req *http.Request, match *RouteMatch) bool {
	for _, route := range r.routes {
		if route.Match(req, match) {
			// Build middleware chain if no error was found
			if match.MatchErr == nil {
				for i := len(r.middlewares) - 1; i >= 0; i-- {
					match.Handler = r.middlewares[i].Middleware(match.Handler)
				}
			}
			return true
		}
	}

	if match.MatchErr == ErrMethodMismatch {
		if r.MethodNotAllowedHandler != nil {
			match.Handler = r.MethodNotAllowedHandler
			return true
		}

		return false
	}

	// Closest match for a router (includes sub-routers)
	if r.NotFoundHandler != nil {
		match.Handler = r.NotFoundHandler
		match.MatchErr = ErrNotFound
		return true
	}

	match.MatchErr = ErrNotFound
	return false
}
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}
func methodNotAllowedHandler() http.Handler { return http.HandlerFunc(methodNotAllowed) }

// // Get registers your function to be called when the given GET path has been requested.
// func (r Router) Get(path string, handler Handler) {
// 	// app.routes.GET = append(app.routes.GET, path)
// 	// app.router.Add(http.MethodGet, path, handler)
// 	// return nil
// }

// Get registers your function to be called when the given GET path has been requested.
// Get registers your function to be called when the given GET path has been requested.
