package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/aerogo/aero"
	"github.com/dtmkeng/bmux/bmux"
)

type route struct {
	method string
	path   string
}

var nullLogger *log.Logger
var loadTestHandler = false

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}
func httpHandlerFunc(w http.ResponseWriter, r *http.Request) {}

func httpHandlerFuncTest(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, r.RequestURI)
}
func (m *mockResponseWriter) WriteHeader(int) {}
func init() {

	runtime.GOMAXPROCS(1)

	// makes logging 'webscale' (ignores them)
	log.SetOutput(new(mockResponseWriter))
	nullLogger = log.New(new(mockResponseWriter), "", 0)

}

// aero
func aeroHandler(c aero.Context) error {
	return nil
}

func aeroHandlerWrite(ctx aero.Context) error {
	io.WriteString(ctx.Response().Internal(), ctx.Get("name"))
	return nil
}
func aeroHandlerTest(ctx aero.Context) error {
	io.WriteString(ctx.Response().Internal(), ctx.Request().Path())
	return nil
}
func loadAero(routes []route) http.Handler {
	var h aero.Handler = aeroHandler
	if loadTestHandler {
		h = aeroHandlerTest
	}
	app := aero.New()
	for _, r := range routes {
		switch r.method {
		case "GET":
			app.Get(r.path, h)
		default:

		}
	}
	return app
}
func loadAeroSingle(method, path string, h aero.Handler) http.Handler {
	app := aero.New()
	switch method {
	case "GET":
		app.Get(path, h)
	default:
		panic("Unknow HTTP method: " + method)
	}
	// }
	return app
}

// bmux
func bmuxHandler(c bmux.Context) error {
	return nil
}

func bmuxHandlerWrite(ctx bmux.Context) error {
	io.WriteString(ctx.Response().Internal(), ctx.Get("name"))
	return nil
}
func bmuxHandlerTest(ctx bmux.Context) error {
	io.WriteString(ctx.Response().Internal(), ctx.Request().Path())
	return nil
}
func loadBmux(routes []route) http.Handler {
	var h bmux.Handler = bmuxHandler
	if loadTestHandler {
		h = bmuxHandlerTest
	}
	app := bmux.NewRouter()
	for _, r := range routes {
		switch r.method {
		case "GET":
			app.Get(r.path, h)
		default:
			// panic("Unknow HTTP method: " + r.method)
		}
	}
	return app
}
func loadBmuxSingle(method, path string, h bmux.Handler) http.Handler {
	app := bmux.NewRouter()
	switch method {
	case "GET":
		app.Get(path, h)
	default:
		panic("Unknow HTTP method: " + method)
	}
	// }
	return app
}

func main() {
	fmt.Println("Usage: go test -bench=. -timeout=20m")
	os.Exit(1)
}
