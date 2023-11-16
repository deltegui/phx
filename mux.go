package phx

import (
	"net/http"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/hash"
	"github.com/deltegui/phx/validator"
)

type phxHandler struct {
	method string
	innery map[string]http.HandlerFunc
}

func (handler phxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	inner, ok := handler.innery[req.Method]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	inner(w, req)
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

type Mux struct {
	injector    *Injector
	router      *http.ServeMux
	endpoints   map[string]phxHandler
	middlewares []Middleware
}

func bootstrap(inj *Injector) {
	inj.Add(func() core.Hasher { return hash.BcryptHasher{} })
	inj.Add(func() core.Validator {
		val := validator.New()
		return func(t interface{}) map[string]string {
			ss, err := val.Validate(t)
			if err != nil {
				panic(err)
			}
			if len(ss) == 0 {
				return nil
			}
			return validator.ModelError(ss)
		}
	})
}

func NewMux() Mux {
	mux := NewMuxEmpty()
	bootstrap(mux.injector)
	return mux
}

func NewMuxEmpty() Mux {
	return Mux{
		injector:    NewInjector(),
		router:      http.DefaultServeMux,
		middlewares: make([]Middleware, 0),
		endpoints:   make(map[string]phxHandler),
	}
}

func NewMuxFrom(other Mux) Mux {
	return Mux{
		injector:    other.injector,
		router:      http.DefaultServeMux,
		middlewares: other.middlewares,
		endpoints:   make(map[string]phxHandler),
	}
}

func (mux *Mux) Add(builder Builder) {
	mux.injector.Add(builder)
}

func (mux *Mux) ShowAvailableBuilders() {
	mux.injector.ShowAvailableBuilders()
}

func (mux *Mux) PopulateStruct(s interface{}) {
	mux.injector.PopulateStruct(s)
}

func (mux *Mux) Use(middleware Middleware) {
	mux.middlewares = append(mux.middlewares, middleware)
}

func (mux *Mux) endpoint(method string, pattern string, builder Builder, middlewares ...Middleware) {
	inner := mux.injector.ResolveHandler(builder)
	handler, ok := mux.endpoints[pattern]
	if !ok {
		handler = phxHandler{
			method: method,
			innery: make(map[string]http.HandlerFunc),
		}
		mux.endpoints[pattern] = handler
		mux.router.Handle(pattern, handler)
	}

	for i := 0; i < len(middlewares); i++ {
		inner = middlewares[i](inner)
	}

	for i := 0; i < len(mux.middlewares); i++ {
		inner = mux.middlewares[i](inner)
	}

	handler.innery[method] = inner
}

func (mux *Mux) Mount(pattern string, inner *Mux) {
	mux.router.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		inner.router.ServeHTTP(w, req)
	})
}

func (mux *Mux) Any(pattern string, builder Builder, middlewares ...Middleware) {
	mux.router.HandleFunc(pattern, mux.injector.ResolveHandler(builder))
}

func (mux *Mux) Get(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodGet, pattern, builder, middlewares...)
}

func (mux *Mux) Post(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodPost, pattern, builder, middlewares...)
}

func (mux *Mux) Patch(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodPatch, pattern, builder, middlewares...)
}

func (mux *Mux) Delete(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodDelete, pattern, builder, middlewares...)
}

func (mux *Mux) Head(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodHead, pattern, builder, middlewares...)
}

func (mux *Mux) Options(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodOptions, pattern, builder, middlewares...)
}

func (mux *Mux) Put(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodPut, pattern, builder, middlewares...)
}

func (mux *Mux) Trace(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodTrace, pattern, builder, middlewares...)
}
