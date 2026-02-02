package router

// HandlerFunc defines the handler function type
type HandlerFunc func(ctx any)

// RadixRouter is a Radix tree based router with parameter support
type RadixRouter struct {
	root *node
}

type nodeType uint8

const (
	static   nodeType = iota // default
	param                    // :param
	catchAll                 // *param
)

type node struct {
	path      string
	indices   string
	children  []*node
	handlers  map[string]HandlerFunc // method -> handler
	priority  uint32
	nType     nodeType
	paramName string // parameter name for :param or *param nodes
}

// NewRadixRouter creates a new router
func NewRadixRouter() *RadixRouter {
	return &RadixRouter{
		root: &node{
			handlers: make(map[string]HandlerFunc),
		},
	}
}

// Add adds a route
func (r *RadixRouter) Add(method, path string, handler HandlerFunc) {
	if path[0] != '/' {
		panic("path must begin with '/'")
	}
	r.root.addRoute(method, path, handler)
}

// Find finds a handler for the given method and path
func (r *RadixRouter) Find(method, path string) (HandlerFunc, map[string]string) {
	if r.root == nil {
		return nil, nil
	}
	handler, params := r.root.getValue(method, path)
	return handler, params
}

func (n *node) addRoute(method, path string, handler HandlerFunc) {
	fullPath := path

	// Empty tree
	if n.path == "" && len(n.children) == 0 {
		n.insertChild(method, path, handler)
		n.nType = static
		return
	}

	for {
		// Find the longest common prefix
		i := longestCommonPrefix(path, n.path)

		// Split edge
		if i < len(n.path) {
			child := &node{
				path:     n.path[i:],
				indices:  n.indices,
				children: n.children,
				handlers: n.handlers,
				priority: n.priority - 1,
				nType:    n.nType,
			}

			n.children = []*node{child}
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
			n.handlers = make(map[string]HandlerFunc)
			n.nType = static
		}

		// Make new node a child of this node
		if i < len(path) {
			path = path[i:]

			if n.nType == param {
				n.priority++
				continue
			}

			idxc := path[0]

			// '/' after param
			if n.nType == param && idxc == '/' && len(n.children) == 1 {
				n = n.children[0]
				n.priority++
				continue
			}

			// Check if a child with the next path byte exists
			childFound := false
			for i, c := range []byte(n.indices) {
				if c == idxc {
					n.priority++
					n = n.children[i]
					childFound = true
					break
				}
			}
			if childFound {
				continue
			}

			// Otherwise insert it
			if idxc != ':' && idxc != '*' {
				n.indices += string([]byte{idxc})
				child := &node{}
				n.addChild(child)
				n = child
			}
			var _ string = fullPath
			n.insertChild(method, path, handler)
			return
		}

		// Otherwise add handler to current node
		if n.handlers == nil {
			n.handlers = make(map[string]HandlerFunc)
		}
		n.handlers[method] = handler
		return
	}
}

func (n *node) insertChild(method, path string, handler HandlerFunc) {
	for {
		// Find wildcard
		wildcard, i, valid := findWildcard(path)
		if i < 0 { // No wildcard found
			break
		}

		// The wildcard name must not contain ':' and '*'
		if !valid {
			panic("only one wildcard per path segment is allowed")
		}

		// Check if the wildcard has a name
		if len(wildcard) < 2 {
			panic("wildcards must be named")
		}

		// param
		if wildcard[0] == ':' {
			// Insert prefix before the current wildcard
			if i > 0 {
				n.path = path[:i]
				path = path[i:]
			}

			child := &node{
				nType:     param,
				path:      wildcard,
				paramName: wildcard[1:],
			}
			n.addChild(child)
			n = child
			n.priority++

			// If the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &node{
					priority: 1,
				}
				n.addChild(child)
				n = child
				continue
			}

			// Otherwise we're done
			if n.handlers == nil {
				n.handlers = make(map[string]HandlerFunc)
			}
			n.handlers[method] = handler
			return
		}

		// catchAll
		if i+len(wildcard) != len(path) {
			panic("catch-all routes are only allowed at the end of the path")
		}

		if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
			// Insert prefix before the current wildcard
			n.path = path[:i]

			child := &node{
				nType:     catchAll,
				path:      wildcard,
				paramName: wildcard[1:],
				handlers:  map[string]HandlerFunc{method: handler},
				priority:  1,
			}
			n.addChild(child)
			return
		}

		panic("catch-all conflicts with existing handle for the path segment")
	}

	// If no wildcard was found, simply insert the path and handler
	n.path = path
	if n.handlers == nil {
		n.handlers = make(map[string]HandlerFunc)
	}
	n.handlers[method] = handler
}

func (n *node) addChild(child *node) {
	if n.children == nil {
		n.children = make([]*node, 0, 1)
	}
	n.children = append(n.children, child)
}

func (n *node) getValue(method, path string) (HandlerFunc, map[string]string) {
	var params map[string]string

	for {
		prefix := n.path

		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				// Try all the non-wildcard children
				idxc := path[0]
				childFound := false
				for i, c := range []byte(n.indices) {
					if c == idxc {
						n = n.children[i]
						childFound = true
						break
					}
				}
				if childFound {
					continue
				}

				// Check if we have wildcard children
				if len(n.children) > 0 {
					lastChild := n.children[len(n.children)-1]

					if lastChild.nType != static {
						// Use the wildcard child
						n = lastChild

						// We need a new instance of params
						if params == nil {
							params = make(map[string]string)
						}

						switch n.nType {
						case param:
							// Find end (either '/' or path end)
							end := 0
							for end < len(path) && path[end] != '/' {
								end++
							}

							// Save param value
							params[n.paramName] = path[:end]

							// Continue with remaining path
							if end < len(path) {
								if len(n.children) > 0 {
									path = path[end:]
									n = n.children[0]
									continue
								}

								// ... but we can't
								return nil, nil
							}

							if handler := n.handlers[method]; handler != nil {
								return handler, params
							}

							return nil, nil

						case catchAll:
							params[n.paramName] = path

							if handler := n.handlers[method]; handler != nil {
								return handler, params
							}

							return nil, nil

						default:
							panic("invalid node type")
						}
					}
				}

				// No wildcard children either
				return nil, nil
			}
		}

		// No match
		if path != prefix {
			return nil, nil
		}

		// We should have reached the node containing the handler
		if handler := n.handlers[method]; handler != nil {
			return handler, params
		}

		return nil, nil
	}
}

// Find the wildcard and check validation
func findWildcard(path string) (wildcard string, i int, valid bool) {
	// Find start
	for start, c := range []byte(path) {
		// A wildcard starts with ':' (param) or '*' (catch-all)
		if c != ':' && c != '*' {
			continue
		}

		// Find end
		valid = true
		for end, c := range []byte(path[start+1:]) {
			switch c {
			case '/':
				return path[start : start+1+end], start, valid
			case ':', '*':
				valid = false
			}
		}
		return path[start:], start, valid
	}
	return "", -1, false
}

func longestCommonPrefix(a, b string) int {
	i := 0
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}
