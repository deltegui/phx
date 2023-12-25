package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/deltegui/phx"
)

const (
	CorsAny string = "*"
)

var corsDefaultOptions CorsOptions = CorsOptions{
	AllowOrigin:  CorsAny,
	AllowMethods: []string{CorsAny},
	AllowHeaders: []string{CorsAny},
	MaxAge:       864000,
}

type CorsOptions struct {
	AllowOrigin  string
	AllowMethods []string
	AllowHeaders []string
	MaxAge       int
}

func CorsDefault() phx.Middleware {
	return Cors(corsDefaultOptions)
}

func Cors(opt CorsOptions) phx.Middleware {
	isOriginAllowed := func(origin string) bool {
		if len(opt.AllowOrigin) == 0 || opt.AllowOrigin == CorsAny {
			return true
		}
		reqOrigin := origin
		if len(reqOrigin) == 0 || reqOrigin != opt.AllowOrigin {
			return false
		}
		return true
	}

	isAllHeadersAllowed := func(headers []string) bool {
		if len(opt.AllowHeaders) == 0 || len(opt.AllowHeaders) == 1 && opt.AllowHeaders[0] == CorsAny {
			return true
		}
	next:
		for _, rh := range headers {
			for _, ah := range opt.AllowHeaders {
				if rh == ah {
					continue next
				}
			}
			return false
		}
		return true
	}

	isMethodAllowed := func(method string) bool {
		if len(opt.AllowMethods) == 0 || (len(opt.AllowMethods) == 1 && opt.AllowMethods[0] == CorsAny) {
			return true
		}
		for _, allowed := range opt.AllowMethods {
			if allowed == method {
				return true
			}
		}
		return false
	}

	getHeadersNames := func(ctx *phx.Context) []string {
		flatten := make([]string, len(ctx.Req.Header))
		i := 0
		for key := range ctx.Req.Header {
			flatten[i] = key
			i++
		}
		return flatten
	}

	return func(next phx.Handler) phx.Handler {
		return func(ctx *phx.Context) error {
			if ctx.Req.Method == http.MethodOptions {
				ctx.Res.Header().Add("Access-Control-Allow-Origin", opt.AllowOrigin)
				ctx.Res.Header().Add("Access-Control-Allow-Methods", strings.Join([]string(opt.AllowMethods), ", "))
				ctx.Res.Header().Add("Access-Control-Max-Age", strconv.Itoa(opt.MaxAge))
				ctx.Res.Header().Add("Access-Control-Allow-Credentials", "true")

				reqMethod := ctx.Req.Header.Get("Access-Control-Request-Method")
				if len(reqMethod) > 0 && !isMethodAllowed(reqMethod) {
					return ctx.Forbidden("Method not allowed by CORS preflight: %s", reqMethod)
				}

				reqHeaders := ctx.Req.Header.Get("Access-Control-Request-Headers")
				if len(reqHeaders) > 0 && !isAllHeadersAllowed(strings.Split(reqHeaders, ", ")) {
					return ctx.Forbidden("Request headers not allowed")
				}

				return ctx.NotContent()
			}
			if !isMethodAllowed(ctx.Req.Method) {
				return ctx.Forbidden("Method not allowed by CORS: %s", ctx.Req.Method)
			}
			origin := ctx.Req.Header.Get("Origin")
			if len(origin) != 0 && !isOriginAllowed(origin) {
				return ctx.Forbidden("Origin not allowed by CORS")
			}
			if !isAllHeadersAllowed(getHeadersNames(ctx)) {
				return ctx.Forbidden("Request headers not allowed")
			}
			err := next(ctx)
			ctx.Res.Header().Add("Access-Control-Allow-Origin", opt.AllowOrigin)
			return err
		}
	}
}
