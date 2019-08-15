package bmux

import (
	"compress/gzip"
	stdContext "context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/akyoto/color"
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
	Vars    map[string]string

	// MatchErr is set to appropriate matching error
	// It is set to ErrMethodMismatch if there is a mismatch in
	// the request method and route method
	MatchErr error
}

// Application represents a single web service.
type Application struct {
	Config *Configuration
	// Sessions              session.Manager
	// Security              ApplicationSecurity
	// Linters               []Linter
	// ContentSecurityPolicy *csp.ContentSecurityPolicy

	router     Routers
	routeTests map[string][]string
	start      time.Time
	rewrite    []func(RewriteContext)
	// middleware     []Middleware
	// pushConditions []func(Context) bool
	onStart    []func()
	onShutdown []func()
	// onPush         []func(Context)
	onError        []func(Context, error)
	stop           chan os.Signal
	pushOptions    http.PushOptions
	contextPool    sync.Pool
	gzipWriterPool sync.Pool
	serversMutex   sync.Mutex
	servers        [2]*http.Server

	routes struct {
		GET []string
	}
	// Configurable Handler to be used when no route matches.
	NotFoundHandler http.Handler

	// Configurable Handler to be used when the request method does not match the route.
	MethodNotAllowedHandler http.Handler

	// Routes to be matched, in order.
	// routes []*Route

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
}

// NewRouter creates a new application.
func NewRouter() *Application {
	app := &Application{namedRoutes: make(map[string]*Route)}

	// Default CSP
	// Context pool
	app.contextPool.New = func() interface{} {
		return &context{
			app: app,
		}
	}

	// Push options describes the headers that are sent
	// to our server to retrieve the push response.
	app.pushOptions = http.PushOptions{
		Method: "GET",
		Header: http.Header{
			acceptEncodingHeader: []string{"gzip"},
		},
	}

	return app
}

// // NewRouter returns a new router instance.
// func NewRouter() *Router {
// 	return &Router{namedRoutes: make(map[string]*Route)}
// }

// Router bmxu
type Router struct {
	// Configurable Handler to be used when no route matches.
	NotFoundHandler http.Handler

	// Configurable Handler to be used when the request method does not match the route.
	MethodNotAllowedHandler http.Handler

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

// func (r *Application) ServeHTTP(w http.ResponseWriter, req *http.Request) {
// 	if !r.skipClean {
// 		path := req.URL.Path
// 		if r.useEncodedPath {
// 			path = req.URL.EscapedPath()
// 		}
// 		// Clean path to canonical form and redirect.
// 		if p := cleanPath(path); p != path {

// 			// Added 3 lines (Philip Schlump) - It was dropping the query string and #whatever from query.
// 			// This matches with fix in go 1.2 r.c. 4 for same problem.  Go Issue:
// 			// http://code.google.com/p/go/issues/detail?id=5252
// 			url := *req.URL
// 			url.Path = p
// 			p = url.String()

// 			w.Header().Set("Location", p)
// 			w.WriteHeader(http.StatusMovedPermanently)
// 			return
// 		}
// 	}
// 	fmt.Println(req.URL.Path)
// 	var match RouteMatch
// 	var handler http.Handler
// 	// if r.Match(req, &match) {
// 	// 	handler = match.Handler
// 	// 	req = setVars(req, match.Vars)
// 	// 	req = setCurrentRoute(req, match.Route)
// 	// }

// 	if handler == nil && match.MatchErr == ErrMethodMismatch {
// 		handler = methodNotAllowedHandler()
// 	}

// 	if handler == nil {
// 		handler = http.NotFoundHandler()
// 	}

// 	handler.ServeHTTP(w, req)
// }

// NewContext ...
func (app *Application) NewContext(req *http.Request, res http.ResponseWriter) *context {
	ctx := app.contextPool.Get().(*context)
	ctx.status = http.StatusOK
	ctx.request.inner = req
	ctx.response.inner = res
	ctx.session = nil
	ctx.paramCount = 0
	ctx.modifierCount = 0
	return ctx
}

// ServeHTTP responds to the given request.
func (app *Application) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	ctx := app.NewContext(request, response)

	for _, rewrite := range app.rewrite {
		rewrite(ctx)
	}

	app.router.Lookup(request.Method, request.URL.Path, ctx)

	if ctx.handler == nil {
		response.WriteHeader(http.StatusNotFound)
		ctx.Close()
		return
	}

	err := ctx.handler(ctx)

	if err != nil {
		for _, callback := range app.onError {
			callback(ctx, err)
		}
	}

	ctx.Close()
}
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

// acquireGZipWriter will return a clean gzip writer from the pool.
func (app *Application) acquireGZipWriter(response io.Writer) *gzip.Writer {
	var writer *gzip.Writer
	obj := app.gzipWriterPool.Get()

	if obj == nil {
		writer, _ = gzip.NewWriterLevel(response, gzip.BestCompression)
		return writer
	}

	writer = obj.(*gzip.Writer)
	writer.Reset(response)
	return writer
}

// Run starts your application.
func (app *Application) Run() {
	// app.BindMiddleware()
	app.ListenAndServe()

	for _, callback := range app.onStart {
		callback()
	}

	// app.TestRoutes()
	app.wait()
	app.Shutdown()
}

// Use adds middleware to your middleware chain.
// func (app *Application) Use(middlewares ...Middleware) {
// 	app.middleware = append(app.middleware, middlewares...)
// }

// Load loads the application configuration from config.json.
// func (app *Application) Load() {
// 	config, err := LoadConfig("config.json")

// 	if err != nil {
// 		// Ignore missing config file, we can perfectly run without one
// 		return
// 	}

// 	app.Config = config
// }

// BindMiddleware applies the middleware to every router node.
// This is called by `Run` automatically and should never be called
// outside of tests.
// func (app *Application) BindMiddleware() {
// 	app.router.Each(func(node *tree) {
// 		if node.data != nil {
// 			node.data = node.data.Bind(app.middleware...)
// 		}
// 	})
// }

// ListenAndServe starts the server.
// It guarantees that a TCP listener is listening on the ports defined in the config
// when the function returns.
func (app *Application) ListenAndServe() {
	// if app.Security.Key != "" && app.Security.Certificate != "" {
	// listener := app.listen(":" + strconv.Itoa(app.Config.Ports.HTTPS))
	// go app.serveHTTPS(listener)
	// fmt.Println("Server running on:", color.GreenString("https://localhost:"+strconv.Itoa(app.Config.Ports.HTTPS)))
	// }

	listener := app.listen(":" + strconv.Itoa(8080))
	go app.serveHTTP(listener)
	fmt.Println("Server running on:", color.GreenString("http://localhost:"+strconv.Itoa(8080)))
}

// wait will make the process wait until it is killed.
func (app *Application) wait() {
	<-app.stop
}

// Shutdown will gracefully shut down all servers.
func (app *Application) Shutdown() {
	app.serversMutex.Lock()
	defer app.serversMutex.Unlock()

	shutdown(app.servers[0])
	shutdown(app.servers[1])

	for _, callback := range app.onShutdown {
		callback()
	}
}

// OnStart registers a callback to be executed on server start.
func (app *Application) OnStart(callback func()) {
	app.onStart = append(app.onStart, callback)
}

// OnEnd registers a callback to be executed on server shutdown.
func (app *Application) OnEnd(callback func()) {
	app.onShutdown = append(app.onShutdown, callback)
}

// // OnPush registers a callback to be executed when an HTTP/2 push happens.
// func (app *Application) OnPush(callback func(Context)) {
// 	app.onPush = append(app.onPush, callback)
// }

// / listen returns a Listener for the given address.
func (app *Application) listen(address string) Listener {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		panic(err)
	}

	return Listener{listener.(*net.TCPListener)}
}

// serveHTTP serves requests from the given listener.
func (app *Application) serveHTTP(listener Listener) {
	server := app.createServer()

	app.serversMutex.Lock()
	app.servers[0] = server
	app.serversMutex.Unlock()

	// This will block the calling goroutine until the server shuts down.
	// The returned error is never nil and in case of a normal shutdown
	// it will be `http.ErrServerClosed`.
	err := server.Serve(listener)

	if err != http.ErrServerClosed {
		panic(err)
	}
}

// createServer creates an http server instance.
func (app *Application) createServer() *http.Server {
	return &http.Server{
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      180 * time.Second,
		IdleTimeout:       120 * time.Second,
		// TLSConfig:         createTLSConfig(),
	}
}

// // serveHTTPS serves requests from the given listener.
// func (app *Application) serveHTTPS(listener Listener) {
// 	server := app.createServer()

// 	app.serversMutex.Lock()
// 	app.servers[1] = server
// 	app.serversMutex.Unlock()

// 	// This will block the calling goroutine until the server shuts down.
// 	// The returned error is never nil and in case of a normal shutdown
// 	// it will be `http.ErrServerClosed`.
// 	err := server.ServeTLS(listener, app.Security.Certificate, app.Security.Key)

// 	if err != http.ErrServerClosed {
// 		panic(err)
// 	}
// }

// shutdown will gracefully shut down the server.
func shutdown(server *http.Server) {
	if server == nil {
		return
	}

	// Add a timeout to the server shutdown
	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 250*time.Millisecond)
	defer cancel()

	// Shut down server
	err := server.Shutdown(ctx)

	if err != nil {
		fmt.Println(err)
	}
}

// Get registers your function to be called when the given GET path has been requested.
func (app *Application) Get(path string, handler Handler) {
	app.routes.GET = append(app.routes.GET, path)
	app.router.Add(http.MethodGet, path, handler)
}
