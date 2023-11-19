package phx

import (
	"context"
	"net/http"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/csrf"
	"github.com/deltegui/phx/session"
)

type contextKey int

const authenticatedUserKey contextKey = 0

type SessionAuth struct {
	sessionManager *session.Manager
	redirect       bool
	redirectURL    string
}

func NewSessionAuth(sessionManager *session.Manager) SessionAuth {
	return SessionAuth{
		sessionManager: sessionManager,
		redirect:       false,
		redirectURL:    "",
	}
}

func NewSessionAuthWithRedirection(sessionManager *session.Manager, redirectURL string) SessionAuth {
	return SessionAuth{
		sessionManager: sessionManager,
		redirect:       true,
		redirectURL:    redirectURL,
	}
}

func (authMiddle SessionAuth) Authorize(next Handler) Handler {
	return func(ctx *Context) {
		user, err := authMiddle.sessionManager.ReadSessionCookie(ctx.Req)
		if err != nil {
			authMiddle.handleError(ctx)
			return
		}
		ctx.Req = makeRequestWithUser(ctx.Req, user)
		next(ctx)
	}
}

func (authMiddle SessionAuth) Admin(next Handler) Handler {
	return func(ctx *Context) {
		user, err := authMiddle.sessionManager.ReadSessionCookie(ctx.Req)
		if err != nil {
			authMiddle.handleError(ctx)
			return
		}
		if user.Role != core.RoleAdmin {
			authMiddle.handleError(ctx)
			return
		}
		ctx.Req = makeRequestWithUser(ctx.Req, user)
		next(ctx)
	}
}

func (authMiddle SessionAuth) AuthorizeRoles(roles []core.Role, next Handler) Handler {
	return func(ctx *Context) {
		user, err := authMiddle.sessionManager.ReadSessionCookie(ctx.Req)
		if err != nil {
			authMiddle.handleError(ctx)
			return
		}
		for _, authorizedRol := range roles {
			if user.Role == authorizedRol {
				next(ctx)
				return
			}
		}
		authMiddle.handleError(ctx)
	}
}

func makeRequestWithUser(req *http.Request, user session.User) *http.Request {
	ctxWithUser := context.WithValue(req.Context(), authenticatedUserKey, user)
	return req.WithContext(ctxWithUser)
}

func getUser(req *http.Request) session.User {
	return req.Context().Value(authenticatedUserKey).(session.User)
}

func (authMiddle SessionAuth) handleError(ctx *Context) {
	if authMiddle.redirect {
		http.Redirect(ctx.Res, ctx.Req, authMiddle.redirectURL, http.StatusTemporaryRedirect)
	} else {
		ctx.Res.WriteHeader(http.StatusUnauthorized)
	}
}

func csrfMiddleware(csrf *csrf.Csrf) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) {
			if ctx.Req.Method != http.MethodPost {
				next(ctx)
				return
			}
			if csrf.CheckRequest(ctx.Req) {
				next(ctx)
				return
			}
			ctx.Res.WriteHeader(http.StatusForbidden)
		}
	}
}

func corsMiddleware(methods, origin string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) {
			next(ctx)
			header := ctx.Res.Header()
			header.Set("Access-Control-Allow-Methods", methods)
			header.Set("Access-Control-Allow-Origin", origin)
		}
	}
}
