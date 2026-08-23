package server

import (
	"net/http"
	"strings"
)

type route struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// Router is a minimal case-insensitive path router with an optional fallback.
type Router struct {
	routes []route
	// NotFound handles unmatched requests (e.g. static file serving).
	NotFound http.HandlerFunc
}

// NewRouter creates an empty router.
func NewRouter() *Router {
	return &Router{}
}

// Handle registers a handler for method+path.
func (r *Router) Handle(method, path string, h http.HandlerFunc) {
	r.routes = append(r.routes, route{method: strings.ToUpper(method), path: strings.ToLower(path), handler: h})
}

// ServeHTTP dispatches to the matching handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := strings.ToLower(req.URL.Path)
	method := strings.ToUpper(req.Method)
	for _, rt := range r.routes {
		if rt.method == method && rt.path == path {
			rt.handler(w, req)
			return
		}
	}
	// wildcard method (e.g. ANY) matches any HTTP method
	for _, rt := range r.routes {
		if rt.method == "ANY" && rt.path == path {
			rt.handler(w, req)
			return
		}
	}
	// method not matched but path exists -> 404 per Java behavior
	for _, rt := range r.routes {
		if rt.path == path {
			write404(w)
			return
		}
	}
	if r.NotFound != nil {
		r.NotFound(w, req)
		return
	}
	write404(w)
}

func write404(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"code":404,"message":"404 Not Found !","data":null}`))
}