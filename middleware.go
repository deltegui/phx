package phx

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/csrf"
	"github.com/deltegui/phx/session"
)

type auth interface {
	authorize() Middleware
	admin() Middleware
	authorizeRoles(roles []core.Role) Middleware
}

type contextKey int

const authenticatedUserKey contextKey = 0

type sessionAuth struct {
	sessionManager *session.Manager
	redirect       bool
	redirectURL    string
}

func newSessionAuth(sessionManager *session.Manager) sessionAuth {
	return sessionAuth{
		sessionManager: sessionManager,
		redirect:       false,
		redirectURL:    "",
	}
}

func newSessionAuthWithRedirection(sessionManager *session.Manager, redirectURL string) sessionAuth {
	return sessionAuth{
		sessionManager: sessionManager,
		redirect:       true,
		redirectURL:    redirectURL,
	}
}

func (authMiddle sessionAuth) authorize() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			user, err := authMiddle.sessionManager.ReadSessionCookie(ctx.Req)
			if err != nil {
				authMiddle.handleError(ctx)
				return nil
			}
			ctx.Req = makeRequestWithUser(ctx.Req, user)
			return next(ctx)
		}
	}
}

func (authMiddle sessionAuth) admin() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			user, err := authMiddle.sessionManager.ReadSessionCookie(ctx.Req)
			if err != nil {
				authMiddle.handleError(ctx)
				return nil
			}
			if user.Role != core.RoleAdmin {
				authMiddle.handleError(ctx)
				return nil
			}
			ctx.Req = makeRequestWithUser(ctx.Req, user)
			return next(ctx)
		}
	}
}

func (authMiddle sessionAuth) authorizeRoles(roles []core.Role) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			user, err := authMiddle.sessionManager.ReadSessionCookie(ctx.Req)
			if err != nil {
				authMiddle.handleError(ctx)
				return nil
			}
			for _, authorizedRol := range roles {
				if user.Role == authorizedRol {
					return next(ctx)
				}
			}
			authMiddle.handleError(ctx)
			return nil
		}
	}
}

func makeRequestWithUser(req *http.Request, user session.User) *http.Request {
	ctxWithUser := context.WithValue(req.Context(), authenticatedUserKey, user)
	return req.WithContext(ctxWithUser)
}

func getUser(req *http.Request) session.User {
	return req.Context().Value(authenticatedUserKey).(session.User)
}

func (authMiddle sessionAuth) handleError(ctx *Context) {
	if authMiddle.redirect {
		http.Redirect(ctx.Res, ctx.Req, authMiddle.redirectURL, http.StatusTemporaryRedirect)
		log.Printf("Authentication failed. Redirecting to url: %s", authMiddle.redirectURL)
	} else {
		ctx.Res.WriteHeader(http.StatusUnauthorized)
		log.Println("Authentication failed")
	}
}

func csrfMiddleware(csrf *csrf.Csrf) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if ctx.Req.Method == http.MethodGet || ctx.Req.Method == http.MethodOptions {
				return next(ctx)
			}
			if csrf.CheckRequest(ctx.Req) {
				return next(ctx)
			}
			ctx.Res.WriteHeader(http.StatusForbidden)
			return fmt.Errorf("Expired csrf token")
		}
	}
}

func corsMiddleware(methods, origin string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if err := next(ctx); err != nil {
				return err
			}
			header := ctx.Res.Header()
			header.Set("Access-Control-Allow-Methods", methods)
			header.Set("Access-Control-Allow-Origin", origin)
			return nil
		}
	}
}

func HttpLogMiddleware(next Handler) Handler {
	return func(ctx *Context) error {
		log.Printf(
			"[PHX] request from %s (%s) to (%s) %s",
			ctx.Req.RemoteAddr,
			ctx.Req.UserAgent(),
			ctx.Req.Method,
			ctx.Req.RequestURI)
		return next(ctx)
	}
}
