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
	"github.com/deltegui/phx/pagination"
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
	validate core.Validator
	sessions *session.Manager
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
	sessions    *session.Manager
	auth        auth
}

type CorsConfig struct {
	AllowMethods string
	AllowOrigin  string
}

func (r *Router) UseTemplate(fs embed.FS) {
	r.tmplFS = fs
}

func (r *Router) UseLocalization(files embed.FS, sharedKey, errorsKey string) {
	var cypher core.Cypher
	cypher = r.injector.GetByType(reflect.TypeOf(&cypher).Elem()).(core.Cypher)
	loc := localizer.NewLocalizerStore(files, sharedKey, errorsKey, cypher)
	r.locstore = &loc
}

func (r *Router) UseCsrf(expires time.Duration) {
	var cypher core.Cypher
	cypher = r.injector.GetByType(reflect.TypeOf(&cypher).Elem()).(core.Cypher)
	r.csrf = csrf.New(expires, cypher)
	r.middlewares = append(r.middlewares, csrfMiddleware(r.csrf))
}

func (r *Router) UseCors(methods, origin string) {
	r.router.GlobalOPTIONS = preflightCorsHanlder(methods, origin)
	r.middlewares = append(r.middlewares, corsMiddleware(methods, origin))
}

func (r *Router) UseStatic(path string) {
	r.router.NotFound = http.FileServer(http.Dir(path))
}

func (r *Router) UseStaticEmbedded(fs embed.FS) {
	r.router.NotFound = http.FileServer(http.FS(fs))
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
		sessions: r.sessions,
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

func (r *Router) UseSession(provider session.SessionStore, duration time.Duration) {
	var hasher core.Hasher
	hasher = r.injector.GetByType(reflect.TypeOf(&hasher).Elem()).(core.Hasher)
	var cypher core.Cypher
	cypher = r.injector.GetByType(reflect.TypeOf(&cypher).Elem()).(core.Cypher)
	r.sessions = session.NewManager(provider, hasher, duration, cypher)
}

func (r *Router) UseSessionAuth() {
	if r.sessions == nil {
		log.Panicln("[PHX] Cannot use session authorization if you dont enable sessions. Please call to router's method 'UseSession' or any variant beofre calling 'UseSessisonAuth'")
	}
	r.auth = newSessionAuth(r.sessions)
}

func (r *Router) UseSessionAuthWithRedirection(redirectUrl string) {
	if r.sessions == nil {
		log.Panicln("[PHX] Cannot use session authorization if you dont enable sessions. Please call to router's method 'UseSession' or any variant beofre calling 'UseSessisonAuth'")
	}
	r.auth = newSessionAuthWithRedirection(r.sessions, redirectUrl)
}

func (r *Router) ensureHaveAuth() {
	if r.auth == nil {
		log.Panicln("[PHX] Cannot use Authorize middleware if you dont provide an authorization implementation. Use 'UseSesssionAuth' to enable authorization")
	}
}

func (r *Router) Authorize() Middleware {
	r.ensureHaveAuth()
	return r.auth.authorize()
}

func (r *Router) Admin() Middleware {
	r.ensureHaveAuth()
	return r.auth.admin()
}

func (r *Router) AuthorizeRoles(roles []core.Role) Middleware {
	r.ensureHaveAuth()
	return r.auth.authorizeRoles(roles)
}

func (r *Context) CreateSessionCookie(user session.User) {
	r.sessions.CreateSessionCookie(r.Res, user)
}

func (r *Context) ReadSessionCookie() (session.User, error) {
	return r.sessions.ReadSessionCookie(r.Req)
}

func (r *Context) DestroySession() error {
	return r.sessions.DestroySession(r.Res, r.Req)
}

func (ctx *Context) GetUser() session.User {
	return getUser(ctx.Req)
}

func (ctx *Context) HaveSession() bool {
	_, err := ctx.ReadSessionCookie()
	return err == nil
}

func (ctx *Context) PaginationToVM(pag pagination.Pagination) pagination.ViewModel {
	return pagination.PaginationToVM(pag, ctx.GetLocalizer("common/pagination"))
}

func (ctx *Context) GetLocalizer(file string) localizer.Localizer {
	return ctx.locstore.GetUsingRequest(file, ctx.Req)
}

func (ctx *Context) Localize(file, key string) string {
	return ctx.locstore.GetUsingRequest(file, ctx.Req).Get(key)
}

func (ctx *Context) LocalizeError(err core.UseCaseError) string {
	return ctx.locstore.GetLocalizedError(err, ctx.Req)
}

func (ctx *Context) LocalizeWithoutShared(file, key string) string {
	return ctx.locstore.GetUsingRequestWithoutShared(file, ctx.Req).Get(key)
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

// PrintLogo takes a embedded filesystem and file path and prints your fancy ascii logo.
// It will fail if your file is not found.
func PrintLogoEmbedded(fs embed.FS, path string) {
	logo, err := fs.ReadFile(path)
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

func (ctx *Context) GetUrlParam(name string) string {
	return ctx.params.ByName(name)
}

func (ctx *Context) GetQueryParam(name string) string {
	return ctx.Req.URL.Query().Get(name)
}

func (ctx *Context) GetCurrentLanguage() string {
	return ctx.locstore.ReadCookie(ctx.Req)
}

func (ctx *Context) ChangeLanguage(to string) {
	ctx.locstore.CreateCookie(ctx.Res, to)
}

func (ctx *Context) Validate(s any) map[string]string {
	return ctx.validate(s)
}

func (ctx *Context) ParseJson(dst any) error {
	return json.NewDecoder(ctx.Req.Body).Decode(dst)
}
