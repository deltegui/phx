package extensions

import (
	"embed"
	"time"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/csrf"
	"github.com/deltegui/phx/cypher"
	"github.com/deltegui/phx/middleware"
	"github.com/deltegui/phx/renderer"
	"github.com/deltegui/phx/session"
)

func AddCypherWithPassword(r *phx.Router, password string) {
	r.Add(func() core.Cypher {
		return cypher.NewWithPasswordAsString(password)
	})
}

func AddCypher(r *phx.Router) {
	r.Add(func() core.Cypher { return cypher.New() })
}

func UseCsrf(r *phx.Router, duration time.Duration) {
	r.Add(func(cy core.Cypher) *csrf.Csrf {
		return csrf.New(duration, cy)
	})
	r.Run(func(c *csrf.Csrf) {
		r.Use(middleware.Csrf(c))
	})
}

func AddSession(r *phx.Router, duration time.Duration) {
	r.Add(func(hasher core.Hasher, cy core.Cypher) *session.Manager {
		return session.NewManager(
			session.NewMemoryStore(),
			hasher,
			duration,
			cy)
	})
}

func AddSessionWithStore(r *phx.Router, duration time.Duration, store session.SessionStore) {
	r.Add(func(hasher core.Hasher, cy core.Cypher) *session.Manager {
		return session.NewManager(
			store,
			hasher,
			duration,
			cy)
	})
}

type Authorization struct {
	redirect string
	manager  *session.Manager
}

func AddAuthorizationWithRedirect(r *phx.Router, redirect string) Authorization {
	auth := Authorization{}
	r.Run(func(manager *session.Manager) {
		auth.manager = manager
	})
	auth.redirect = redirect
	return auth
}

func AddAuthorization(r *phx.Router) Authorization {
	return AddAuthorizationWithRedirect(r, "")
}

func (auth Authorization) Authorize() phx.Middleware {
	return middleware.Authorize(auth.manager, auth.redirect)
}

func (auth Authorization) Roles(roles []core.Role) phx.Middleware {
	return middleware.AuthorizeRoles(auth.manager, auth.redirect, roles)
}

func (auth Authorization) Admin() phx.Middleware {
	return middleware.Admin(auth.manager, auth.redirect)
}

func AddRendering(r *phx.Router, fs embed.FS) *renderer.TemplateRenderer {
	rend := renderer.NewTemplateRenderer(fs)
	rend.AddDefaultTemplateFunctions()
	r.Add(func() phx.Renderer { return rend })
	r.Add(func() *renderer.TemplateRenderer { return rend })
	return rend
}

func UseCors(r *phx.Router, opt middleware.CorsOptions) {
	r.Use(middleware.Cors(opt))
}

func UseCorsDefault(r *phx.Router) {
	r.Use(middleware.CorsDefault())
}
