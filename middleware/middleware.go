package middleware

import (
	"log"

	"github.com/deltegui/phx"
)

func Logger(next phx.Handler) phx.Handler {
	return func(ctx *phx.Context) error {
		log.Printf(
			"[PHX] request from %s (%s) to (%s) %s",
			ctx.Req.RemoteAddr,
			ctx.Req.UserAgent(),
			ctx.Req.Method,
			ctx.Req.RequestURI)
		return next(ctx)
	}
}
