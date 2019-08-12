package bmux

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/aerogo/session"
)

// Handler is a function that deals with the given request/response context.
type Handler func(Context) error

type dataType = Handler
type tree struct {
	prefix    string
	data      dataType
	children  [224]*tree
	parameter *tree
	wildcard  *tree
	kind      byte
}

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

// Context represents the interface for a request & response context.
type Context interface {
	// AddModifier(Modifier)
	// App() *Application
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

// node types
const (
	separator = '/'
	parameter = ':'
	wildcard  = '*'
)

// controlFlow tells the main loop what it should do next.
type controlFlow int

// controlFlow values.
const (
	controlStop  controlFlow = 0
	controlBegin controlFlow = 1
	controlNext  controlFlow = 2
)

// Modifier ...
type Modifier = func([]byte) []byte
type request struct {
	inner *http.Request
}
type response struct {
	inner http.ResponseWriter
}

// context represents a request & response context.
type contexts struct {
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

// NewContext ...
func (app *Application) NewContext(req *http.Request, res http.ResponseWriter) *contexts {
	ctx := app.contextPool.Get().(*contexts)
	ctx.status = http.StatusOK
	ctx.request.inner = req
	ctx.response.inner = res
	ctx.session = nil
	ctx.paramCount = 0
	ctx.modifierCount = 0
	return ctx
}

// Path returns the relative request path, e.g. /blog/post/123.
func (ctx *contexts) Path() string {
	return ctx.request.inner.URL.Path
}

// SetPath sets the relative request path, e.g. /blog/post/123.
func (ctx *contexts) SetPath(path string) {
	ctx.request.inner.URL.Path = path
}
func contextGet(r *http.Request, key interface{}) interface{} {
	return r.Context().Value(key)
}

func contextSet(r *http.Request, key, val interface{}) *http.Request {
	if val == nil {
		return r
	}

	return r.WithContext(context.WithValue(r.Context(), key, val))
}

// dataType specifies which type of data we are going to save for each node.

// add adds a new element to the tree.
func (node *tree) add(path string, data dataType) {
	// Search tree for equal parts until we can no longer proceed
	i := 0
	offset := 0

	for {
	begin:
		switch node.kind {
		case parameter:
			// This only occurs when the same parameter based route is added twice.
			// node: /post/:id|
			// path: /post/:id|
			if i == len(path) {
				node.data = data
				return
			}

			// When we hit a separator, we'll search for a fitting child.
			if path[i] == separator {
				var control controlFlow
				node, offset, control = node.end(path, data, i, offset)

				switch control {
				case controlStop:
					return
				case controlBegin:
					goto begin
				case controlNext:
					goto next
				}
			}

		default:
			if i == len(path) {
				// The path already exists.
				// node: /blog|
				// path: /blog|
				if i-offset == len(node.prefix) {
					node.data = data
					return
				}

				// The path ended but the node prefix is longer.
				// node: /blog|feed
				// path: /blog|
				node.split(i-offset, "", data)
				return
			}

			// The node we just checked is entirely included in our path.
			// node: /|
			// path: /|blog
			if i-offset == len(node.prefix) {
				var control controlFlow
				node, offset, control = node.end(path, data, i, offset)

				switch control {
				case controlStop:
					return
				case controlBegin:
					goto begin
				case controlNext:
					goto next
				}
			}

			// We got a conflict.
			// node: /b|ag
			// path: /b|riefcase
			if path[i] != node.prefix[i-offset] {
				node.split(i-offset, path[i:], data)
				return
			}
		}

	next:
		i++
	}
}
func (node *tree) split(index int, path string, data dataType) {
	// Create split node with the remaining string
	splitNode := node.clone(node.prefix[index:])

	/// The existing data must be removed
	node.reset(node.prefix[:index])

	// If the path is empty, it means we don't create a 2nd child node.
	// Just assign the data for the existing node and store a single child node.
	if path == "" {
		node.data = data
		node.children[splitNode.prefix[0]-32] = splitNode
		return
	}

	node.children[splitNode.prefix[0]-32] = splitNode

	// Create new nodes with the remaining path
	node.append(path, data)
}

// clone clones the node with a new prefix.
func (node *tree) clone(prefix string) *tree {
	return &tree{
		prefix:    prefix,
		data:      node.data,
		children:  node.children,
		parameter: node.parameter,
		wildcard:  node.wildcard,
		kind:      node.kind,
	}
}

// reset resets the existing node data.
func (node *tree) reset(prefix string) {
	node.prefix = prefix
	node.data = nil
	node.parameter = nil
	node.wildcard = nil
	node.kind = 0
	node.children = [224]*tree{}
}

// append appends the given path to the tree.
func (node *tree) append(path string, data dataType) {
	// At this point, all we know is that somewhere
	// in the remaining string we have parameters.
	// node: /user|
	// path: /user|/:userid
	for {
		if path == "" {
			node.data = data
			return
		}

		paramStart := strings.IndexByte(path, parameter)

		if paramStart == -1 {
			paramStart = strings.IndexByte(path, wildcard)
		}

		// If it's a static route we are adding,
		// just add the remainder as a normal node.
		if paramStart == -1 {
			// If the node itself doesn't have a prefix (root node),
			// don't add a child and use the node itself.
			if node.prefix == "" {
				node.prefix = path
				node.data = data
				return
			}

			child := &tree{
				prefix: path,
				data:   data,
			}

			node.children[path[0]-32] = child
			child.addTrailingSlash(data)
			return
		}

		// If we're directly in front of a parameter,
		// add a parameter node.
		if paramStart == 0 {
			paramEnd := strings.IndexByte(path, separator)

			if paramEnd == -1 {
				paramEnd = len(path)
			}

			child := &tree{
				prefix: path[1:paramEnd],
				kind:   path[paramStart],
			}

			switch child.kind {
			case parameter:
				child.addTrailingSlash(data)
				node.parameter = child
				node = child
				path = path[paramEnd:]
				continue

			case wildcard:
				child.data = data
				node.wildcard = child
				return
			}
		}

		// We know there's a parameter, but not directly at the start.

		// If the node itself doesn't have a prefix (root node),
		// don't add a child and use the node itself.
		if node.prefix == "" {
			node.prefix = path[:paramStart]
			path = path[paramStart:]
			continue
		}

		// Add a normal node with the path before the parameter start.
		child := &tree{
			prefix: path[:paramStart],
		}

		// Allow trailing slashes to return
		// the same content as their parent node.
		if child.prefix == "/" {
			child.data = node.data
		}

		node.children[path[0]-32] = child
		node = child
		path = path[paramStart:]
	}
}

// addTrailingSlash adds a trailing slash with the same data.
func (node *tree) addTrailingSlash(data dataType) {
	if strings.HasSuffix(node.prefix, "/") || node.children[separator-32] != nil || node.kind == wildcard {
		return
	}

	node.children[separator-32] = &tree{
		prefix: "/",
		data:   data,
	}
}

// end is called when the node was fully parsed
// and needs to decide the next control flow.
func (node *tree) end(path string, data dataType, i int, offset int) (*tree, int, controlFlow) {
	child := node.children[path[i]-32]

	if child != nil {
		node = child
		offset = i
		return node, offset, controlNext
	}

	// No fitting children found, does this node even contain a prefix yet?
	// If no prefix is set, this is the starting node.
	if node.prefix == "" {
		node.append(path[i:], data)
		return node, offset, controlStop
	}

	// node: /user/|:id
	// path: /user/|:id/profile
	if node.parameter != nil {
		node = node.parameter
		offset = i
		return node, offset, controlBegin
	}

	node.append(path[i:], data)
	return node, offset, controlStop
}

// addParameter adds a new parameter to the context.
func (ctx *contexts) addParameter(name string, value string) {
	ctx.paramNames[ctx.paramCount] = name
	ctx.paramValues[ctx.paramCount] = value
	ctx.paramCount++
}

// find finds the data for the given path and assigns it to ctx.handler, if available.
func (node *tree) find(path string, ctx *contexts) {
	var (
		i                  int
		offset             int
		lastWildcardOffset int
		lastWildcard       *tree
	)

	// Search tree for equal parts until we can no longer proceed
	for {
	begin:
		switch node.kind {
		case parameter:
			if i == len(path) {
				ctx.addParameter(node.prefix, path[offset:i])
				ctx.handler = node.data
				return
			}

			if path[i] == separator {
				ctx.addParameter(node.prefix, path[offset:i])
				node = node.children[separator-32]
				offset = i
				goto next
			}

		default:
			// We reached the end.
			if i == len(path) {
				// node: /blog|
				// path: /blog|
				if i-offset == len(node.prefix) {
					ctx.handler = node.data
					return
				}

				// node: /blog|feed
				// path: /blog|
				ctx.handler = nil
				return
			}

			// The node we just checked is entirely included in our path.
			// node: /|
			// path: /|blog
			if i-offset == len(node.prefix) {
				if node.wildcard != nil {
					lastWildcard = node.wildcard
					lastWildcardOffset = i
				}

				child := node.children[path[i]-32]

				if child != nil {
					node = child
					offset = i
					goto next
				}

				// node: /|:id
				// path: /|blog
				if node.parameter != nil {
					node = node.parameter
					offset = i
					goto begin
				}

				// node: /|*any
				// path: /|image.png
				if node.wildcard != nil {
					ctx.addParameter(node.wildcard.prefix, path[i:])
					ctx.handler = node.wildcard.data
					return
				}

				ctx.handler = nil
				return
			}

			// We got a conflict.
			// node: /b|ag
			// path: /b|riefcase
			if path[i] != node.prefix[i-offset] {
				if lastWildcard != nil {
					ctx.addParameter(lastWildcard.prefix, path[lastWildcardOffset:])
					ctx.handler = lastWildcard.data
					return
				}

				ctx.handler = nil
				return
			}
		}

	next:
		i++
	}
}

// in the ServeHTTP part of the web server.
func (ctx *contexts) Close() {
	ctx.app.contextPool.Put(ctx)
}
