# 🐦 Phx
A little, highly opinated framework to create complete web apps.

Demo:
```go
package main

import (
	"littlecash/locale"
	"littlecash/static"
	"littlecash/views"
	"log"
	"net/http"
	"time"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/cypher"
	"github.com/deltegui/phx/session"
)

func ParseAll(r *phx.Router) {
	r.Parse(views.DemoIndex, "index.html", "demo/index.html")
	r.Parse(views.Login, "layout.html", "account/layout.html", "account/login.html")
	r.Parse(views.Arr, "arr.html", "demo/arr.html")
	r.ShowAvailableTemplates()
}

func main() {
	r := phx.NewRouter()
	r.Bootstrap()
	r.Add(func() core.Cypher {
		csrfpass := "you can generate a password with the framework or let it be random"
		return cypher.NewWithPasswordAsString(csrfpass)
	})
	r.ShowAvailableBuilders()

	r.Use(phx.HttpLogMiddleware)

	r.UseStaticEmbedded(static.Files)
	r.UseTemplate(views.Files)
	r.AddDefaultTemplateFunctions()
	r.UseCsrf(15 * time.Minute)
	r.UseLocalization(locale.Files, locale.Shared, locale.Errors)
	r.UseSessionInMemory(1 * time.Hour)
	r.UseSessionAuthWithRedirection("/login")

	views.ParseAll(r)

	r.Get("/", func() phx.Handler {
		return func(ctx *phx.Context) {
			ctx.StringOK("Hola mundo desde aqui")
		}
	})
	r.Get("/demo", func() phx.Handler {
		return func(ctx *phx.Context) {
			ctx.Render(http.StatusOK, views.DemoIndex, struct{}{})
		}
	}, r.Authorize())
	r.Get("/change/:lang", func() phx.Handler {
		return func(ctx *phx.Context) {
			from := ctx.GetCurrentLanguage()
			to := ctx.GetUrlParam("lang")
			if len(to) == 0 {
				ctx.String(http.StatusBadRequest, "Nothing to change!")
				return
			}
			ctx.ChangeLanguage(to)
			ctx.StringOK("Changed from '%s' to '%s'", from, to)
		}
	})
	r.Post("/demo", func(hasher core.Hasher) phx.Handler {
		type f struct {
			Name     string `validate:"required,min=3,max=255"`
			Password string `validate:"required,min=3,max=255"`
		}
		return func(ctx *phx.Context) {
			var form f
			ctx.ParseForm(&form)
			errs := ctx.Validate(form)
			if errs != nil {
				ctx.RenderWithErrors(http.StatusBadRequest, views.DemoIndex, struct{}{}, errs)
				return
			}
			form.Password = hasher.Hash(form.Password)
			ctx.JsonOK(form)
		}
	}, r.Authorize())
	r.Get("/login", func() phx.Handler {
		return func(ctx *phx.Context) {
			ctx.RenderOK(views.Login, AccountViewModel{})
		}
	})
	r.Post("/login", func() phx.Handler {
		return func(ctx *phx.Context) {
			var vm AccountViewModel
			ctx.ParseForm(&vm)
			if errs := ctx.Validate(vm); errs != nil {
				log.Println("MAL")
				ctx.RenderWithErrors(http.StatusBadRequest, views.Login, vm, errs)
				return
			}
			ctx.CreateSessionCookie(session.User{
				Id:   0,
				Name: vm.Name,
				Role: core.RoleAdmin,
			})
			ctx.Redirect("/demo")
		}
	})
	r.Get("/logout", func() phx.Handler {
		return func(ctx *phx.Context) {
			ctx.DestroySession()
			ctx.Redirect("/login")
		}
	})

	r.Get("/arr", func() phx.Handler {
		return func(ctx *phx.Context) {
			ctx.RenderOK(views.Arr, nil)
		}
	})

	r.Post("/arr", func() phx.Handler {
		return func(ctx *phx.Context) {
			var names []string
			if err := ctx.ParseJson(&names); err != nil {
				ctx.String(http.StatusBadRequest, "Invalid JSON")
				return
			}
			log.Println("OK")
			ctx.JsonOK(names)
		}
	})

	phx.PrintLogoEmbedded(views.Files, "banner.txt")
	r.Run(":3000")
}

type AccountViewModel struct {
	Name           string `validate:"required,min=3,max=255"`
	ErrorMessage   string
	HaveBeenLogout bool
}
```