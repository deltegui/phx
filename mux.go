package phx

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/hash"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/validator"
	"github.com/julienschmidt/httprouter"
)

type Middleware func(Handler) Handler

type Handler func(c *Context) error

type Renderer interface {
	Render(ctx *Context, status int, parsed string, vm interface{}) error
	RenderWithErrors(ctx *Context, status int, parsed string, vm interface{}, formErrors map[string]string) error
}

type Router struct {
	injector     *Injector
	router       *httprouter.Router
	middlewares  []Middleware
	ErrorHandler func(*Context, error)

	locstore *localizer.LocalizerStore
	validate core.Validator
}

func (r *Router) UseLocalization(files embed.FS, sharedKey, errorsKey string) {
	var cypher core.Cypher
	instance, err := r.injector.GetByType(reflect.TypeOf(&cypher).Elem())
	if err != nil {
		panic("A type of core.Cypher is needed to use localization. Register it in the injector.")
	}
	cypher, ok := instance.(core.Cypher)
	if !ok {
		panic("Expected that the registered dependency for type core.Cypher would be compatible with interface core.Cypher")
	}
	loc := localizer.NewLocalizerStore(files, sharedKey, errorsKey, cypher)
	r.locstore = &loc
}

func (r *Router) Static(path string) {
	r.router.NotFound = http.FileServer(http.Dir(path))
}

func (r *Router) StaticEmbedded(fs embed.FS) {
	r.router.NotFound = http.FileServer(http.FS(fs))
}

func (r *Router) StaticMount(url, path string) {
	r.router.ServeFiles(fmt.Sprintf("%s/*filepath", url), http.Dir(path))
}

func (r *Router) StaticMountEmbedded(url string, fs embed.FS) {
	r.router.ServeFiles(fmt.Sprintf("%s/*filepath", url), http.FS(fs))
}

func defaultErrorHandler(ctx *Context, err error) {
	log.Println("[PHX] Error:", err)
	ctx.InternalServerError(err.Error())
}

func NewRouter() *Router {
	return &Router{
		injector:     NewInjector(),
		router:       httprouter.New(),
		middlewares:  []Middleware{},
		ErrorHandler: defaultErrorHandler,
		validate:     validator.New(),
	}
}

func NewRouterFromOther(r *Router) *Router {
	return &Router{
		injector:     r.injector,
		router:       httprouter.New(),
		ErrorHandler: r.ErrorHandler,
		middlewares:  r.middlewares,
		locstore:     r.locstore,
	}
}

func (r *Router) createContext(w http.ResponseWriter, req *http.Request, params httprouter.Params) *Context {
	ctx := &Context{
		Req:      req,
		Res:      w,
		params:   params,
		locstore: r.locstore,
		validate: r.validate,
		ctx:      context.Background(),
	}
	instance, err := r.injector.Get(&ctx.renderer)
	if err != nil {
		return ctx
	}
	rend, ok := instance.(Renderer)
	if !ok {
		log.Println("Expected injetro's registered renderer to be of type 'Renderer', but it is other type")
		return ctx
	}
	ctx.renderer = rend
	return ctx
}

func (r *Router) Bootstrap() {
	r.injector.Add(func() core.Hasher { return hash.BcryptHasher{} })
}

func (r *Router) Add(builder Builder) {
	r.injector.Add(builder)
}

func (r *Router) Run(runner Runner) {
	r.injector.Run(runner)
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

func (r Router) Listen(address string) {
	server := http.Server{
		Addr:    address,
		Handler: r.router,
	}
	go startServer(&server)
	waitAndStopServer(&server)
}

func Redirect(to string) func() Handler {
	return func() Handler {
		return func(c *Context) error {
			http.RedirectHandler(to, http.StatusTemporaryRedirect).ServeHTTP(c.Res, c.Req)
			return nil
		}
	}
}
