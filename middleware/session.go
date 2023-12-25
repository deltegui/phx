package middleware

import (
	"log"
	"net/http"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/session"
)

func Authorize(manager *session.Manager, url string) phx.Middleware {
	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			user, err := manager.ReadSessionCookie(ctx.Req)
			if err != nil {
				handleError(ctx, url)
				return nil
			}
			ctx.Set(session.ContextKey, user)
			return next(ctx)
		}
	}
}

func AuthorizeRoles(manager *session.Manager, url string, roles []core.Role) phx.Middleware {
	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			user, err := manager.ReadSessionCookie(ctx.Req)
			if err != nil {
				handleError(ctx, url)
				return nil
			}
			for _, authorizedRol := range roles {
				if user.Role == authorizedRol {
					return next(ctx)
				}
			}
			handleError(ctx, url)
			return nil
		}
	}
}

func Admin(manager *session.Manager, url string) phx.Middleware {
	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			user, err := manager.ReadSessionCookie(ctx.Req)
			if err != nil {
				handleError(ctx, url)
				return nil
			}
			if user.Role != core.RoleAdmin {
				handleError(ctx, url)
				return nil
			}
			ctx.Set(session.ContextKey, user)
			return next(ctx)
		}
	}
}

func handleError(ctx *phx.Context, url string) {
	if len(url) > 0 {
		http.Redirect(ctx.Res, ctx.Req, url, http.StatusTemporaryRedirect)
		log.Printf("Authentication failed. Redirecting to url: %s", url)
	} else {
		ctx.Res.WriteHeader(http.StatusUnauthorized)
		log.Println("Authentication failed")
	}
}
