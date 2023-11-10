package middleware

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

func (authMiddle SessionAuth) Authorize(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, err := authMiddle.sessionManager.ReadSessionCookie(req)
		if err != nil {
			authMiddle.handleError(w, req)
			return
		}
		next(w, makeRequestWithUser(req, user))
	})
}

func (authMiddle SessionAuth) Admin(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, err := authMiddle.sessionManager.ReadSessionCookie(req)
		if err != nil {
			authMiddle.handleError(w, req)
			return
		}
		if user.Role != core.RoleAdmin {
			authMiddle.handleError(w, req)
			return
		}
		next(w, makeRequestWithUser(req, user))
	})
}

func (authMiddle SessionAuth) AuthorizeRoles(roles []core.Role, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, err := authMiddle.sessionManager.ReadSessionCookie(req)
		if err != nil {
			authMiddle.handleError(w, req)
			return
		}
		for _, authorizedRol := range roles {
			if user.Role == authorizedRol {
				next(w, req)
				return
			}
		}
		authMiddle.handleError(w, req)
	})
}

func makeRequestWithUser(req *http.Request, user session.User) *http.Request {
	ctxWithUser := context.WithValue(req.Context(), authenticatedUserKey, user)
	return req.WithContext(ctxWithUser)
}

func GetUser(req *http.Request) session.User {
	return req.Context().Value(authenticatedUserKey).(session.User)
}

func (authMiddle SessionAuth) handleError(w http.ResponseWriter, req *http.Request) {
	if authMiddle.redirect {
		http.Redirect(w, req, authMiddle.redirectURL, http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func Csrf(csrf csrf.Csrf) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodPost {
				next.ServeHTTP(w, req)
				return
			}
			if csrf.CheckRequest(req) {
				next.ServeHTTP(w, req)
				return
			}
			w.WriteHeader(http.StatusForbidden)
		})
	}
}
