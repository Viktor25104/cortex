package gin

import (
	"encoding/json"
	"net/http"
	"strings"
)

type HandlerFunc func(*Context)

type H map[string]interface{}

type route struct {
	method  string
	path    string
	handler HandlerFunc
}

type Engine struct {
	routes []route
}

func Default() *Engine {
	return &Engine{routes: make([]route, 0)}
}

func (e *Engine) POST(path string, handler HandlerFunc) {
	e.addRoute(http.MethodPost, path, handler)
}

func (e *Engine) GET(path string, handler HandlerFunc) {
	e.addRoute(http.MethodGet, path, handler)
}

func (e *Engine) addRoute(method, path string, handler HandlerFunc) {
	e.routes = append(e.routes, route{method: method, path: path, handler: handler})
}

func (e *Engine) Run(addr string) error {
	return http.ListenAndServe(addr, e)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rt := range e.routes {
		if rt.method != r.Method {
			continue
		}
		params, ok := matchRoute(rt.path, r.URL.Path)
		if !ok {
			continue
		}
		ctx := &Context{Writer: w, Request: r, params: params}
		rt.handler(ctx)
		return
	}
	http.NotFound(w, r)
}

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	params  map[string]string
}

func (c *Context) ShouldBindJSON(obj interface{}) error {
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(obj)
}

func (c *Context) JSON(status int, obj interface{}) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	encoder := json.NewEncoder(c.Writer)
	_ = encoder.Encode(obj)
}

func (c *Context) Param(key string) string {
	if c.params == nil {
		return ""
	}
	return c.params[key]
}

func matchRoute(pattern, path string) (map[string]string, bool) {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)
	for i := range patternParts {
		pp := patternParts[i]
		pv := pathParts[i]
		if strings.HasPrefix(pp, ":") {
			params[pp[1:]] = pv
			continue
		}
		if pp != pv {
			return nil, false
		}
	}
	return params, true
}
