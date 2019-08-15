package bmux

import (
	"fmt"
	"net/http"
)

// Routers ..
type Routers struct {
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

// selectTree returns the tree by the given HTTP method.
func (router *Routers) selectTree(method string) *tree {
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

// Add registers a new handler for the given method and path.
func (router *Routers) Add(method string, path string, handler Handler) {
	tree := router.selectTree(method)

	if tree == nil {
		panic(fmt.Errorf("Unknown HTTP method: '%s'", method))
	}

	tree.add(path, handler)
}
