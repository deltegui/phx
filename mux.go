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
	"strings"
	"syscall"
	"time"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/validator"
)

type Middleware func(Handler) Handler

type Handler func(c *Context) error

type Renderer interface {
	Render(ctx *Context, status int, parsed string, vm interface{}) error
	RenderWithErrors(ctx *Context, status int, parsed string, vm interface{}, formErrors map[string]string) error
}

type Router struct {
	injector     *Injector
	router       *http.ServeMux
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

func exactUrl(method, path string) string {
	var endPath string
	if strings.HasSuffix(path, "/") {
		endPath = "{$}"
	} else {
		endPath = "/{$}"
	}
	return fmt.Sprintf("%s %s%s", method, path, endPath)
}

func (r *Router) Static(path string) {
	r.StaticMount("GET /", path)
}

func (r *Router) StaticEmbedded(fs embed.FS) {
	r.StaticMountEmbedded("GET /", fs)
}

func (r *Router) StaticMount(url, path string) {
	server := http.FileServer(http.Dir(path))
	r.staticServer(url, server)
}

func (r *Router) StaticMountEmbedded(url string, fs embed.FS) {
	server := http.FileServerFS(fs)
	r.staticServer(url, server)
}

func (r *Router) staticServer(url string, server http.Handler) {
	endUrl := ""
	if !strings.HasSuffix(url, "/") {
		endUrl = "/"
	}
	full := fmt.Sprintf("GET %s%s", url, endUrl)
	r.router.Handle(full, http.StripPrefix(url, server))
}

func defaultErrorHandler(ctx *Context, err error) {
	log.Println("[PHX] Error:", err)
	ctx.InternalServerError(err.Error())
}

func NewRouter() *Router {
	return &Router{
		injector:     NewInjector(),
		router:       http.NewServeMux(),
		middlewares:  []Middleware{},
		ErrorHandler: defaultErrorHandler,
		validate:     validator.New(),
	}
}

func NewRouterFromOther(r *Router) *Router {
	return &Router{
		injector:     r.injector,
		router:       http.NewServeMux(),
		ErrorHandler: r.ErrorHandler,
		middlewares:  r.middlewares,
		locstore:     r.locstore,
	}
}

func (r *Router) createContext(w http.ResponseWriter, req *http.Request) *Context {
	ctx := &Context{
		Req:      req,
		Res:      w,
		locstore: r.locstore,
		validate: r.validate,
		ctx:      context.Background(),
	}

	var rend Renderer
	instance, err := r.injector.GetByType(reflect.TypeOf(&rend).Elem())
	if err != nil {
		return ctx
	}
	rend, ok := instance.(Renderer)
	if !ok {
		log.Println("Expected injetor's registered renderer to be of type 'Renderer', but it is other type")
		return ctx
	}
	ctx.renderer = rend

	var cy core.Cypher
	cyInstance, err := r.injector.GetByType(reflect.TypeOf(&cy).Elem())
	if err != nil {
		return ctx
	}
	cy, ok = cyInstance.(core.Cypher)
	if !ok {
		log.Println("Expected injetor's registered cypher to be of type 'core.Cypher', but it is other type")
		return ctx
	}
	ctx.cy = cy

	return ctx
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
	muxPattern := exactUrl(method, pattern)
	r.router.HandleFunc(muxPattern, func(w http.ResponseWriter, req *http.Request) {
		ctx := r.createContext(w, req)
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
