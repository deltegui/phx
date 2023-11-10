package phx

import (
	"net/http"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/hash"
	"github.com/deltegui/phx/validator"
)

type phxHandler struct {
	method string
	inner  http.HandlerFunc
}

func (handler phxHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != handler.method {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	handler.inner(w, req)
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

type Mux struct {
	Injector   *Injector
	router     *http.ServeMux
	middleware []Middleware
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
	bootstrap(mux.Injector)
	return mux
}

func NewMuxEmpty() Mux {
	return Mux{
		Injector:   NewInjector(),
		router:     http.DefaultServeMux,
		middleware: make([]Middleware, 0),
	}
}

func (mux *Mux) Use(middleware Middleware) {
	mux.middleware = append(mux.middleware, middleware)
}

func (mux *Mux) endpoint(method string, pattern string, builder Builder, middlewares ...Middleware) {
	inner := mux.Injector.ResolveHandler(builder)
	handler := phxHandler{
		method: method,
		inner:  inner,
	}
	mux.router.Handle(pattern, handler)
}

func (mux *Mux) Mount(pattern string, inner *Mux) {
	mux.router.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		inner.router.ServeHTTP(w, req)
	})
}

func (mux *Mux) Any(pattern string, builder Builder, middlewares ...Middleware) {
	mux.router.HandleFunc(pattern, mux.Injector.ResolveHandler(builder))
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

func (mux *Mux) Connect(pattern string, builder Builder, middlewares ...Middleware) {
	mux.endpoint(http.MethodConnect, pattern, builder, middlewares...)
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
