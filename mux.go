package phx

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/csrf"
	"github.com/deltegui/phx/hash"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/persistence"
	"github.com/deltegui/phx/session"
	"github.com/deltegui/phx/validator"
	"github.com/jmoiron/sqlx"
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
	validate core.Validator
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
	validate    core.Validator
	Sessions    *session.Manager
}

type CorsConfig struct {
	AllowMethods string
	AllowOrigin  string
}

func (r *Router) UseTemplate(fs embed.FS) {
	r.tmplFS = fs
}

func (r *Router) UseLocalization(files embed.FS, sharedKey, errorsKey string) {
	loc := localizer.NewLocalizerStore(files, sharedKey, errorsKey)
	r.locstore = &loc
}

func (r *Router) UseCsrf(expires time.Duration) {
	r.csrf = csrf.New(expires)
	r.middlewares = append(r.middlewares, csrfMiddleware(r.csrf))
}

func (r *Router) UseCors(methods, origin string) {
	r.router.GlobalOPTIONS = preflightCorsHanlder(methods, origin)
	r.middlewares = append(r.middlewares, corsMiddleware(methods, origin))
}

func (r *Router) UseStatic(path string) {
	r.router.NotFound = http.FileServer(http.Dir(path))
}

func NewRouter() *Router {
	return &Router{
		injector:    NewInjector(),
		router:      httprouter.New(),
		middlewares: []Middleware{},
		tmpl:        make(map[string]*template.Template),
		validate:    validator.New(),
	}
}

func preflightCorsHanlder(methods, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Access-Control-Request-Method") != "" {
			header := w.Header()
			header.Set("Access-Control-Allow-Methods", methods)
			header.Set("Access-Control-Allow-Origin", origin)
		}
		w.WriteHeader(http.StatusNoContent)
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
		validate: r.validate,
	}
}

func (r *Router) Bootstrap() {
	r.injector.Add(func() core.Hasher { return hash.BcryptHasher{} })
}

func (r *Router) Add(builder Builder) {
	r.injector.Add(builder)
}

func (r *Router) ShowAvailableBuilders() {
	r.injector.ShowAvailableBuilders()
}

func (r *Router) ShowAvailableTemplates() {
	log.Println("Templates:")
	for key, value := range r.tmpl {
		log.Println(key, "->", value.Tree.Name)
	}
}

func (r *Router) PopulateStruct(s interface{}) {
	r.injector.PopulateStruct(s)
}

func (r *Router) UseSessionInMemory(duration time.Duration) {
	r.UseSession(session.NewMemoryStore(), duration)
}

func (r *Router) UseSessionPostgres(db *sqlx.DB, duration time.Duration) {
	r.UseSession(persistence.NewSessionStore(db), duration)
}

func (r *Router) UseSession(provider session.SessionStore, duration time.Duration) {
	var hasher core.Hasher
	hasher, ok := r.injector.Get(reflect.TypeOf(&hasher)).(core.Hasher)
	if !ok {
		log.Panicln("[PHX] Cannot use session if you dont procvide a core hasher implementation. Call Bootstrap method or register a implementation into the dependency injection container")
	}
	r.Sessions = session.NewManager(provider, hasher, duration)
}

func (r *Router) Use(middleware Middleware) {
	r.middlewares = append(r.middlewares, middleware)
}

func (r *Router) Handle(method, pattern string, builder Builder, middlewares ...Middleware) {
	h := r.injector.ResolveHandler(builder)
	for _, m := range r.middlewares {
		h = m(h)
	}
	for _, m := range middlewares {
		h = m(h)
	}
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

func Redirect(to string) func() Handler {
	return func() Handler {
		return func(c *Context) {
			http.RedirectHandler(to, http.StatusTemporaryRedirect).ServeHTTP(c.Res, c.Req)
		}
	}
}

func (ctx *Context) Redirect(to string) {
	http.Redirect(ctx.Res, ctx.Req, to, http.StatusTemporaryRedirect)
}

func (ctx *Context) RedirectCode(to string, code int) {
	http.Redirect(ctx.Res, ctx.Req, to, code)
}

func (ctx *Context) GetUser() session.User {
	return getUser(ctx.Req)
}

func (ctx *Context) GetParam(name string) string {
	return ctx.params.ByName(name)
}

func (ctx *Context) GetCurrentLanguage() string {
	return localizer.ReadCookie(ctx.Req)
}

func (ctx *Context) ChangeLanguage(to string) {
	localizer.CreateCookie(ctx.Res, to)
}

func (ctx *Context) Validate(s any) map[string]string {
	return ctx.validate(s)
}

func (ctx *Context) ParseJson(dst any) error {
	return json.NewDecoder(ctx.Req.Body).Decode(dst)
}
