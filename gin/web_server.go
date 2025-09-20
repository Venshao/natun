// Package gin - 轻量级 Web 框架（无第三方依赖）
package gin

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

type H map[string]interface{}

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

func (c *Context) JSON(code int, obj interface{}) {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	_ = json.NewEncoder(c.W).Encode(obj)
}

func (c *Context) Data(code int, contentType string, data []byte) {
	c.W.Header().Set("Content-Type", contentType)
	c.W.WriteHeader(code)
	_, _ = c.W.Write(data)
}

func (c *Context) ShouldBindJSON(obj interface{}) error {
	return json.NewDecoder(c.R.Body).Decode(obj)
}

// --- Engine ---

type handlerFunc func(*Context)

type Engine struct {
	routes map[string]map[string]handlerFunc
	groups []*Group
	static map[string]fs.FS
}

func New() *Engine {
	e := &Engine{
		routes: make(map[string]map[string]handlerFunc),
		static: make(map[string]fs.FS),
	}
	return e
}

func (e *Engine) addRoute(method, path string, handler handlerFunc) {
	if e.routes[method] == nil {
		e.routes[method] = make(map[string]handlerFunc)
	}
	e.routes[method][path] = handler
}

func (e *Engine) GET(path string, handler handlerFunc) {
	e.addRoute("GET", path, handler)
}

func (e *Engine) POST(path string, handler handlerFunc) {
	e.addRoute("POST", path, handler)
}

func (e *Engine) Group(prefix string) *Group {
	g := &Group{prefix: prefix, engine: e}
	e.groups = append(e.groups, g)
	return g
}

func (e *Engine) StaticFS(route string, filesystem fs.FS) {
	e.static[route] = filesystem
	handler := http.StripPrefix(route, http.FileServer(http.FS(filesystem)))
	e.GET(route+"/*filepath", func(c *Context) {
		handler.ServeHTTP(c.W, c.R)
	})
}

func (e *Engine) Run(addr string) error {
	return http.ListenAndServe(addr, e)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := &Context{W: w, R: r}
	method := r.Method
	path := r.URL.Path

	if e.routes[method] != nil {
		if handler, ok := e.routes[method][path]; ok {
			handler(ctx)
			return
		}
		// 支持通配符 *filepath 静态路由
		for route, handler := range e.routes[method] {
			if strings.Contains(route, "/*") {
				prefix := strings.TrimSuffix(route, "/*filepath")
				if strings.HasPrefix(path, prefix) {
					handler(ctx)
					return
				}
			}
		}
	}

	http.NotFound(w, r)
}

// --- Group ---

type Group struct {
	prefix string
	engine *Engine
}

func (g *Group) GET(path string, handler handlerFunc) {
	g.engine.GET(pathJoin(g.prefix, path), handler)
}

func (g *Group) POST(path string, handler handlerFunc) {
	g.engine.POST(pathJoin(g.prefix, path), handler)
}

func pathJoin(prefix, path string) string {
	return strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(path, "/")
}
