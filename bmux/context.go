package bmux

import (
	stdcontext "context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/aerogo/session"
	"github.com/akyoto/stringutils/unsafe"
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
	App() *Application
	Bytes([]byte) error
	Close()
	// CSS(string) error
	Get(string) string
	// GetInt(string) (int, error)
	// Error(int, ...interface{}) error
	// EventStream(stream *EventStream) error
	// File(string) error
	// HasSession() bool
	// HTML(string) error
	// IP() string
	// JavaScript(string) error
	// JSON(interface{}) error
	// Path() string
	// Query(param string) string
	// ReadAll(io.Reader) error
	// Reader(io.Reader) error
	// ReadSeeker(io.ReadSeeker) error
	// Redirect(status int, url string) error
	// RemoteIP() string
	Request() Request
	Response() Response
	// Session() *session.Session
	// SetStatus(int)
	// Status() int
	String(string) error
	// Text(string) error
}

// context represents a request & response context.
type context struct {
	app           *Application
	status        int
	request       request
	response      response
	session       *session.Session
	handler       Handler
	paramNames    [maxParams]string
	paramValues   [maxParams]string
	paramCount    int
	modifiers     [maxModifiers]Modifier
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

// Response returns the HTTP response.
func (ctx *context) Response() Response {
	return &ctx.response
}

// Request returns the HTTP request.
func (ctx *context) Request() Request {
	return &ctx.request
}

// Close frees up resources and is automatically called
// in the ServeHTTP part of the web server.
func (ctx *context) Close() {
	ctx.app.contextPool.Put(ctx)
}

// App returns the application the context occurred in.
func (ctx *context) App() *Application {
	return ctx.app
}

// Path returns the relative request path, e.g. /blog/post/123.
func (ctx *context) Path() string {
	return ctx.request.inner.URL.Path
}

// SetPath sets the relative request path, e.g. /blog/post/123.
func (ctx *context) SetPath(path string) {
	ctx.request.inner.URL.Path = path
}

// Get retrieves an URL parameter.
func (ctx *context) Get(param string) string {
	for i := 0; i < ctx.paramCount; i++ {
		if ctx.paramNames[i] == param {
			return ctx.paramValues[i]
		}
	}

	return ""
}

// Bytes responds either with raw text or gzipped if the
// text length is greater than the gzip threshold. Requires a byte slice.
func (ctx *context) Bytes(body []byte) error {
	// If the request has been canceled by the client, stop.
	if ctx.request.Context().Err() != nil {
		return errors.New("Request interrupted by the client")
	}

	// If we registered any response body modifiers, invoke them.
	if ctx.modifierCount > 0 {
		for i := 0; i < ctx.modifierCount; i++ {
			body = ctx.modifiers[i](body)
		}
	}

	// Small response
	if len(body) < gzipThreshold {
		ctx.response.inner.WriteHeader(ctx.status)
		_, err := ctx.response.inner.Write(body)
		return err
	}

	// ETag generation
	etag := ETag(body)

	// If client cache is up to date, send 304 with no response body.
	clientETag := ctx.request.Header(ifNoneMatchHeader)

	if etag == clientETag {
		ctx.response.inner.WriteHeader(304)
		return nil
	}

	// Set ETag
	header := ctx.response.inner.Header()
	header.Set(etagHeader, etag)

	// Content type
	contentType := header.Get(contentTypeHeader)
	isMediaType := isMedia(contentType)

	// Cache control header
	if isMediaType {
		header.Set(cacheControlHeader, cacheControlMedia)
	} else {
		header.Set(cacheControlHeader, cacheControlAlwaysValidate)
	}

	// No GZip?
	clientSupportsGZip := strings.Contains(ctx.request.Header(acceptEncodingHeader), "gzip")

	if !ctx.app.Config.GZip || !clientSupportsGZip || !canCompress(contentType) {
		header.Set(contentLengthHeader, strconv.Itoa(len(body)))
		ctx.response.inner.WriteHeader(ctx.status)
		_, err := ctx.response.inner.Write(body)
		return err
	}

	// GZip
	header.Set(contentEncodingHeader, contentEncodingGzip)
	ctx.response.inner.WriteHeader(ctx.status)

	// Write response body
	writer := ctx.app.acquireGZipWriter(ctx.response.inner)
	_, err := writer.Write(body)
	writer.Close()

	// Put the writer back into the pool
	ctx.app.gzipWriterPool.Put(writer)

	// Return the error value of the last Write call
	return err
}

// String responds either with raw text or gzipped if the
// text length is greater than the gzip threshold.
func (ctx *context) String(body string) error {
	return ctx.Bytes(unsafe.StringToBytes(body))
}

// isMedia returns whether the given content type is a media type.
func isMedia(contentType string) bool {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return true
	case strings.HasPrefix(contentType, "video/"):
		return true
	case strings.HasPrefix(contentType, "audio/"):
		return true
	default:
		return false
	}
}

// canCompress returns whether the given content type should be compressed via gzip.
func canCompress(contentType string) bool {
	switch {
	case strings.HasPrefix(contentType, "image/") && contentType != contentTypeSVG:
		return false
	case strings.HasPrefix(contentType, "video/"):
		return false
	case strings.HasPrefix(contentType, "audio/"):
		return false
	default:
		return true
	}
}
