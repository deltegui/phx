package middleware

import (
	"errors"
	"net/http"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/csrf"
)

func Csrf(cs *csrf.Csrf) phx.Middleware {
	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			if ctx.Req.Method == http.MethodGet || ctx.Req.Method == http.MethodOptions {
				ctx.Set(csrf.ContextKey, cs.Generate())
				return next(ctx)
			}
			if !cs.CheckRequest(ctx.Req) {
				ctx.Res.WriteHeader(http.StatusForbidden)
				return errors.New("expired csrf token")
			}
			ctx.Set(csrf.ContextKey, cs.Generate())
			return next(ctx)
		}
	}
}
