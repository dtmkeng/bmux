package bmux

import (
	stdcontext "context"
	"io"
	"net/http"

	"github.com/aerogo/session"
)

const (
	// gzipThreshold should be close to the MTU size of a TCP packet.
	// Regarding performance it makes no sense to compress smaller files.
	// Bandwidth can be saved however the savings are minimal for small files
	// and the overhead of compressing can lead up to a 75% reduction
	// in server speed under high load. Therefore in this case
	// we're trying to optimize for performance, not bandwidth.
	gzipThreshold = 1450

	// maxParams defines the maximum number of parameters per route.
	maxParams = 16

	// maxModifiers defines the maximum number of modifiers per context.
	maxModifiers = 4
)

// addParameter adds a new parameter to the context.
func (ctx *context) addParameter(name string, value string) {
	ctx.paramNames[ctx.paramCount] = name
	ctx.paramValues[ctx.paramCount] = value
	ctx.paramCount++
}

// Context represents the interface for a request & response context.
type Context interface {
	// AddModifier(Modifier)
	App() *Router
	Bytes([]byte) error
	Close()
	CSS(string) error
	Get(string) string
	GetInt(string) (int, error)
	Error(int, ...interface{}) error
	// EventStream(stream *EventStream) error
	File(string) error
	HasSession() bool
	HTML(string) error
	IP() string
	JavaScript(string) error
	JSON(interface{}) error
	Path() string
	Query(param string) string
	ReadAll(io.Reader) error
	Reader(io.Reader) error
	ReadSeeker(io.ReadSeeker) error
	Redirect(status int, url string) error
	RemoteIP() string
	// Request() Request
	// Response() Response
	Session() *session.Session
	SetStatus(int)
	Status() int
	String(string) error
	Text(string) error
}

// context represents a request & response context.
type context struct {
	app         *Application
	status      int
	request     request
	response    response
	session     *session.Session
	handler     Handler
	paramNames  [maxParams]string
	paramValues [maxParams]string
	paramCount  int
	// modifiers     [maxModifiers]Modifier
	modifierCount int
}

func contextGet(r *http.Request, key interface{}) interface{} {
	return r.Context().Value(key)
}

func contextSet(r *http.Request, key, val interface{}) *http.Request {
	if val == nil {
		return r
	}

	return r.WithContext(stdcontext.WithValue(r.Context(), key, val))
}
