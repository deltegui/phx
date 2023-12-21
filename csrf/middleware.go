package csrf

import (
	"fmt"
	"net/http"

	"github.com/deltegui/phx"
)

const ContextKey string = "phx-csrf"

func Middleware(csrf *Csrf) phx.Middleware {
	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			if ctx.Req.Method != http.MethodGet && ctx.Req.Method != http.MethodOptions {
				ctx.Res.WriteHeader(http.StatusForbidden)
				return fmt.Errorf("expired csrf token")
			}
			if !csrf.CheckRequest(ctx.Req) {
				ctx.Res.WriteHeader(http.StatusForbidden)
				return fmt.Errorf("expired csrf token")
			}
			ctx.Set(ContextKey, csrf.Generate())
			return next(ctx)
		}
	}
}
