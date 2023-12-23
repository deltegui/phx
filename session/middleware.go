package session

import (
	"log"
	"net/http"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/core"
)

const ContextKey string = "phx_session_auth"

func Authorize(manager Manager) phx.Middleware {
	return authorize(manager, "")
}

func AuthorizeRedirect(manager Manager, url string) phx.Middleware {
	return authorize(manager, url)
}

func authorize(manager Manager, url string) phx.Middleware {
	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			user, err := manager.ReadSessionCookie(ctx.Req)
			if err != nil {
				handleError(ctx, url)
				return nil
			}
			ctx.Set(ContextKey, user)
			return next(ctx)
		}
	}
}

func AuthorizeRoles(manager Manager) func([]core.Role) phx.Middleware {
	return authorizeRoles(manager, "")
}

func AuthorizeRolesRedirect(manager Manager, url string) func([]core.Role) phx.Middleware {
	return authorizeRoles(manager, url)
}

func authorizeRoles(manager Manager, url string) func([]core.Role) phx.Middleware {
	return func(roles []core.Role) phx.Middleware {
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
}

func Admin(manager Manager) phx.Middleware {
	return admin(manager, "")
}

func AdminRedirect(manager Manager, url string) phx.Middleware {
	return admin(manager, url)
}

func admin(manager Manager, url string) phx.Middleware {
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
			ctx.Set(ContextKey, user)
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
