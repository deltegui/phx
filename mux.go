package phx

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/csrf"
	"github.com/deltegui/phx/hash"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/session"
	"github.com/deltegui/phx/validator"
	"github.com/julienschmidt/httprouter"
)

type Middleware func(Handler) Handler

type Handler func(c *Context)

type Context struct {
	Req      *http.Request
	Res      http.ResponseWriter
	tmpl     map[string]*template.Template
	params   httprouter.Params
	locstore *localizer.LocalizerStore
	csrf     *csrf.Csrf
}

type Router struct {
	injector    *Injector
	router      *httprouter.Router
	middlewares []Middleware
	tmpl        map[string]*template.Template
	tmplFuncs   template.FuncMap
	tmplFS      embed.FS
	csrf        *csrf.Csrf
	locstore    *localizer.LocalizerStore
}

type CorsConfig struct {
	AllowMethods string
	AllowOrigin  string
}

type Config struct {
	CsrfExpiration time.Duration
	Localizer      *localizer.LocalizerStore
	Cors           CorsConfig
	EnableCors     bool
	StaticPath     string
	EnableStatic   bool
	EnableCsrf     bool
}

func NewRouterWithConfig(config Config) *Router {
	router := httprouter.New()
	middlewares := []Middleware{}
	if config.EnableCors {
		router.GlobalOPTIONS = preflightCorsHanlder(config.Cors)
		middlewares = append(middlewares, corsMiddleware(config.Cors))
	}
	if config.EnableStatic {
		router.NotFound = http.FileServer(http.Dir(config.StaticPath))
	}
	var csrfInstance *csrf.Csrf = nil
	if config.EnableCsrf {
		*csrfInstance = csrf.New(config.CsrfExpiration)
		middlewares = append(middlewares, csrfMiddleware(*csrfInstance))
	}
	return &Router{
		injector:    NewInjector(),
		router:      httprouter.New(),
		middlewares: middlewares,
		tmpl:        make(map[string]*template.Template),
		csrf:        csrfInstance,
		locstore:    config.Localizer,
	}
}

func preflightCorsHanlder(c CorsConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Access-Control-Request-Method") != "" {
			header := w.Header()
			header.Set("Access-Control-Allow-Methods", c.AllowMethods)
			header.Set("Access-Control-Allow-Origin", c.AllowOrigin)
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func NewRouter() *Router {
	return NewRouterWithConfig(Config{
		CsrfExpiration: 15 * time.Minute,
		EnableCors:     false,
		EnableStatic:   false,
	})
}

func NewRouterFromOther(r *Router) *Router {
	return &Router{
		injector:    r.injector,
		router:      httprouter.New(),
		middlewares: r.middlewares,
		tmpl:        make(map[string]*template.Template),
		csrf:        r.csrf,
		locstore:    r.locstore,
	}
}

func (r *Router) createContext(w http.ResponseWriter, req *http.Request, params httprouter.Params) *Context {
	return &Context{
		Req:      req,
		Res:      w,
		tmpl:     r.tmpl,
		params:   params,
		locstore: r.locstore,
		csrf:     r.csrf,
	}
}

func (r *Router) Bootstrap(inj *Injector) {
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

func (r *Router) Add(builder Builder) {
	r.injector.Add(builder)
}

func (r *Router) ShowAvailableBuilders() {
	r.injector.ShowAvailableBuilders()
}

func (r *Router) PopulateStruct(s interface{}) {
	r.injector.PopulateStruct(s)
}

func (r *Router) Use(middleware Middleware) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *Router) Handle(method, pattern string, builder Builder, middlewares ...Middleware) {
	h := r.injector.ResolveHandler(builder)
	r.router.Handle(method, pattern, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		ctx := r.createContext(w, req, params)
		h(ctx)
	})
}

func (r *Router) Get(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodGet, pattern, builder, middlewares...)
}

func (r *Router) Post(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodPost, pattern, builder, middlewares...)
}

func (r *Router) Patch(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodPatch, pattern, builder, middlewares...)
}

func (r *Router) Delete(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodDelete, pattern, builder, middlewares...)
}

func (r *Router) Head(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodHead, pattern, builder, middlewares...)
}

func (r *Router) Options(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodOptions, pattern, builder, middlewares...)
}

func (r *Router) Put(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodPut, pattern, builder, middlewares...)
}

func (r *Router) Trace(pattern string, builder Builder, middlewares ...Middleware) {
	r.Handle(http.MethodTrace, pattern, builder, middlewares...)
}

// PrintLogo takes a file path and prints your fancy ascii logo.
// It will fail if your file is not found.
func PrintLogo(logoFile string) {
	logo, err := os.ReadFile(logoFile)
	if err != nil {
		log.Fatalf("Cannot read logo file: %s\n", err)
	}
	fmt.Println(string(logo))
}

func startServer(server *http.Server) {
	log.Println("Listening on address: ", server.Addr)
	log.Println("You are ready to GO!")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln("Error while listening: ", err)
	}
}

func waitAndStopServer(server *http.Server) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	log.Print("Server Stopped")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed:%+v", err)
	}

	log.Print("Server exited properly")
}

func (r Router) Run(address string) {
	server := http.Server{
		Addr:    address,
		Handler: r.router,
	}
	go startServer(&server)
	waitAndStopServer(&server)
}

func (ctx *Context) Redirect(to string) func() http.HandlerFunc {
	return func() http.HandlerFunc {
		return http.RedirectHandler(to, http.StatusTemporaryRedirect).ServeHTTP
	}
}

func (ctx *Context) GetUser() session.User {
	return getUser(ctx.Req)
}

func (ctx *Context) GetParam(name string) string {
	return ctx.params.ByName(name)
}
