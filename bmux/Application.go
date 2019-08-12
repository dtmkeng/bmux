package bmux

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aerogo/csp"
	"github.com/aerogo/session"
)

// Application represents a single web service.
type Application struct {
	Sessions session.Manager
	// Security              ApplicationSecurity
	// Linters               []Linter
	ContentSecurityPolicy *csp.ContentSecurityPolicy

	router         Router
	routeTests     map[string][]string
	start          time.Time
	rewrite        []func(RewriteContext)
	middleware     []middleware
	pushConditions []func(Context) bool
	onStart        []func()
	onShutdown     []func()
	onPush         []func(Context)
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
}

// Get ...
func (app *Application) Get(path string, handler Handler) {
	// fmt.Println(app)
	app.routes.GET = append(app.routes.GET, path)
	app.router.Add(http.MethodGet, path, handler)
	// return
}
