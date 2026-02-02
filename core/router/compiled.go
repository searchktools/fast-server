package router

import (
	"strings"
	"sync"
)

// CompiledRouter is a compile-time optimized router with O(1) lookup
type CompiledRouter struct {
	// Static routes: direct map lookup O(1)
	staticRoutes map[string]map[string]HandlerFunc // path -> method -> handler

	// Parameterized routes: optimized tree structure
	paramRoutes *compiledNode

	// Wildcard routes: cached patterns
	wildcardRoutes []*wildcardRoute

	// Route cache for hot paths
	cache sync.Map // "METHOD:PATH" -> cachedResult

	// Statistics
	hits   uint64
	misses uint64
}

type compiledNode struct {
	// Fixed array for common path segments (cache-friendly)
	staticChildren [128]*compiledNode // indexed by first byte

	// Parameter node (e.g., :id)
	paramChild *compiledNode
	paramName  string

	// Handler for this path
	handlers map[string]HandlerFunc

	// Path segment
	segment string
}

type wildcardRoute struct {
	prefix   string
	paramKey string
	handlers map[string]HandlerFunc
}

type cachedResult struct {
	handler HandlerFunc
	params  map[string]string
}

// NewCompiledRouter creates a new compiled router
func NewCompiledRouter() *CompiledRouter {
	return &CompiledRouter{
		staticRoutes:   make(map[string]map[string]HandlerFunc),
		paramRoutes:    &compiledNode{handlers: make(map[string]HandlerFunc)},
		wildcardRoutes: make([]*wildcardRoute, 0),
	}
}

// Add adds a route and compiles it
func (r *CompiledRouter) Add(method, path string, handler HandlerFunc) {
	if path[0] != '/' {
		panic("path must begin with '/'")
	}

	// Classify route type
	if !strings.Contains(path, ":") && !strings.Contains(path, "*") {
		// Static route - O(1) map lookup
		r.addStaticRoute(method, path, handler)
	} else if strings.Contains(path, "*") {
		// Wildcard route
		r.addWildcardRoute(method, path, handler)
	} else {
		// Parameterized route
		r.addParamRoute(method, path, handler)
	}
}

// addStaticRoute adds a static route (fastest path)
func (r *CompiledRouter) addStaticRoute(method, path string, handler HandlerFunc) {
	if r.staticRoutes[path] == nil {
		r.staticRoutes[path] = make(map[string]HandlerFunc)
	}
	r.staticRoutes[path][method] = handler
}

// addParamRoute adds a parameterized route
func (r *CompiledRouter) addParamRoute(method, path string, handler HandlerFunc) {
	segments := strings.Split(path[1:], "/") // Skip leading /
	node := r.paramRoutes

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		if segment[0] == ':' {
			// Parameter segment
			paramName := segment[1:]
			if node.paramChild == nil {
				node.paramChild = &compiledNode{
					paramName: paramName,
					handlers:  make(map[string]HandlerFunc),
				}
			}
			node = node.paramChild
		} else {
			// Static segment - use array index for cache-friendly access
			idx := segment[0]
			if node.staticChildren[idx] == nil {
				node.staticChildren[idx] = &compiledNode{
					segment:  segment,
					handlers: make(map[string]HandlerFunc),
				}
			}
			node = node.staticChildren[idx]
		}
	}

	node.handlers[method] = handler
}

// addWildcardRoute adds a wildcard route
func (r *CompiledRouter) addWildcardRoute(method, path string, handler HandlerFunc) {
	idx := strings.Index(path, "*")
	prefix := path[:idx]
	paramKey := path[idx+1:]

	route := &wildcardRoute{
		prefix:   prefix,
		paramKey: paramKey,
		handlers: make(map[string]HandlerFunc),
	}
	route.handlers[method] = handler
	r.wildcardRoutes = append(r.wildcardRoutes, route)
}

// Find finds a handler with O(1) complexity for static routes
func (r *CompiledRouter) Find(method, path string) (HandlerFunc, map[string]string) {
	// Step 1: Check cache first (hot path optimization)
	cacheKey := method + ":" + path
	if cached, ok := r.cache.Load(cacheKey); ok {
		result := cached.(*cachedResult)
		return result.handler, result.params
	}

	// Step 2: Try static routes (O(1) map lookup)
	if methods, ok := r.staticRoutes[path]; ok {
		if handler, ok := methods[method]; ok {
			// Cache the result
			r.cache.Store(cacheKey, &cachedResult{handler: handler, params: nil})
			return handler, nil
		}
	}

	// Step 3: Try parameterized routes (O(k) where k = path segments)
	if handler, params := r.findParamRoute(method, path); handler != nil {
		// Cache the result
		r.cache.Store(cacheKey, &cachedResult{handler: handler, params: params})
		return handler, params
	}

	// Step 4: Try wildcard routes
	if handler, params := r.findWildcardRoute(method, path); handler != nil {
		return handler, params
	}

	return nil, nil
}

// findParamRoute finds a parameterized route
func (r *CompiledRouter) findParamRoute(method, path string) (HandlerFunc, map[string]string) {
	segments := strings.Split(path[1:], "/")
	node := r.paramRoutes
	params := make(map[string]string, 4) // Pre-allocate for common case

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Try static match first (cache-friendly array access)
		idx := segment[0]
		if child := node.staticChildren[idx]; child != nil && child.segment == segment {
			node = child
			continue
		}

		// Try parameter match
		if node.paramChild != nil {
			params[node.paramChild.paramName] = segment
			node = node.paramChild
			continue
		}

		// No match
		return nil, nil
	}

	if handler, ok := node.handlers[method]; ok {
		return handler, params
	}

	return nil, nil
}

// findWildcardRoute finds a wildcard route
func (r *CompiledRouter) findWildcardRoute(method, path string) (HandlerFunc, map[string]string) {
	for _, route := range r.wildcardRoutes {
		if strings.HasPrefix(path, route.prefix) {
			if handler, ok := route.handlers[method]; ok {
				params := make(map[string]string)
				params[route.paramKey] = path[len(route.prefix):]
				return handler, params
			}
		}
	}
	return nil, nil
}

// Build optimizes the router (pre-warming cache, etc.)
func (r *CompiledRouter) Build() {
	// Pre-warm common routes
	// This could include JIT compilation in future versions

	// TODO: Generate optimized lookup tables
	// TODO: Analyze route patterns and generate specialized code
	// TODO: Use code generation for ultra-fast path matching
}

// Stats returns router statistics
func (r *CompiledRouter) Stats() (hits, misses uint64, hitRate float64) {
	// TODO: Implement atomic counters for stats
	return 0, 0, 0.0
}

// ClearCache clears the route cache
func (r *CompiledRouter) ClearCache() {
	r.cache = sync.Map{}
}
