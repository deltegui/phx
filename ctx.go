package phx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/cypher"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/pagination"
	"github.com/deltegui/phx/session"
)

type Context struct {
	Req    *http.Request
	Res    http.ResponseWriter
	params httprouter.Params
	ctx    context.Context

	locstore *localizer.Store

	renderer Renderer

	validate core.Validator

	cy core.Cypher
}

func (ctx *Context) Set(key, value any) {
	ctx.ctx = context.WithValue(ctx.ctx, key, value)
}

func (ctx *Context) Get(key any) any {
	return ctx.ctx.Value(key)
}

func (ctx *Context) PaginationToVM(pag pagination.Pagination) pagination.ViewModel {
	return pagination.ToVM(pag, ctx.GetLocalizer("common/pagination"))
}

func (ctx *Context) HaveLocalizer() bool {
	return ctx.locstore != nil
}

func (ctx *Context) GetLocalizer(file string) localizer.Localizer {
	return ctx.locstore.GetUsingRequest(file, ctx.Req)
}

func (ctx *Context) Localize(file, key string) string {
	return ctx.locstore.GetUsingRequest(file, ctx.Req).Get(key)
}

func (ctx *Context) LocalizeError(err core.UseCaseError) string {
	return ctx.locstore.GetLocalizedError(err, ctx.Req)
}

func (ctx *Context) LocalizeWithoutShared(file, key string) string {
	return ctx.locstore.GetUsingRequestWithoutShared(file, ctx.Req).Get(key)
}

func (ctx *Context) GetUser() session.User {
	return ctx.Get(session.ContextKey).(session.User)
}

func (ctx *Context) HaveSession() bool {
	instance := ctx.Get(session.ContextKey)
	if instance == nil {
		return false
	}
	if _, ok := instance.(session.User); !ok {
		return false
	}
	return true
}

func (ctx *Context) Redirect(to string) error {
	http.Redirect(ctx.Res, ctx.Req, to, http.StatusTemporaryRedirect)
	return nil
}

func (ctx *Context) RedirectCode(to string, code int) error {
	http.Redirect(ctx.Res, ctx.Req, to, code)
	return nil
}

func (ctx *Context) GetURLParam(name string) string {
	return ctx.params.ByName(name)
}

func (ctx *Context) GetQueryParam(name string) string {
	return ctx.Req.URL.Query().Get(name)
}

func (ctx *Context) GetCurrentLanguage() string {
	return ctx.locstore.ReadCookie(ctx.Req)
}

func (ctx *Context) ChangeLanguage(to string) error {
	return ctx.locstore.CreateCookie(ctx.Res, to)
}

func (ctx *Context) Validate(s any) map[string][]core.ValidationError {
	return ctx.validate(s)
}

func (ctx *Context) ParseJson(dst any) error {
	return json.NewDecoder(ctx.Req.Body).Decode(dst)
}

func (ctx *Context) String(status int, data string, a ...any) error {
	ctx.Res.WriteHeader(status)
	fmt.Fprintf(ctx.Res, data, a...)
	return nil
}

func (ctx *Context) BadRequest(data string, a ...any) error {
	return ctx.String(http.StatusBadRequest, data, a...)
}

func (ctx *Context) NotFound(data string, a ...any) error {
	return ctx.String(http.StatusNotFound, data, a...)
}

func (ctx *Context) Ok(data string, a ...any) error {
	return ctx.String(http.StatusOK, data, a...)
}

func (ctx *Context) InternalServerError(data string, a ...any) error {
	return ctx.String(http.StatusInternalServerError, data, a...)
}

func (ctx *Context) NotContent() error {
	ctx.Res.WriteHeader(http.StatusNoContent)
	return nil
}

func (ctx *Context) Forbidden(data string, a ...any) error {
	return ctx.String(http.StatusForbidden, data, a...)
}

func (ctx *Context) Json(status int, data any) error {
	response, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}
	ctx.Res.WriteHeader(status)
	ctx.Res.Header().Set("Content-Type", "application/json")
	_, err = ctx.Res.Write(response)
	return err
}

func (ctx *Context) JsonOk(data interface{}) error {
	return ctx.Json(http.StatusOK, data)
}

func (ctx *Context) Render(status int, parsed string, vm interface{}) error {
	if ctx.renderer == nil {
		panic("Cannot call render. Missing dependency: phx.Renderer")
	}
	return ctx.renderer.Render(ctx, status, parsed, vm)
}

func (ctx *Context) RenderOk(parsed string, vm interface{}) error {
	return ctx.Render(http.StatusOK, parsed, vm)
}

func (ctx *Context) RenderBlock(status int, parsed, blockName string, vm interface{}) error {
	if ctx.renderer == nil {
		panic("Cannot call render. Missing dependency: phx.Renderer")
	}
	return ctx.renderer.RenderBlock(ctx, status, parsed, blockName, vm)
}

func (ctx *Context) RenderBlockOk(parsed, blockName string, vm interface{}) error {
	return ctx.RenderBlock(http.StatusOK, parsed, blockName, vm)
}

func (ctx *Context) RenderWithErrors(
	status int,
	parsed string,
	vm interface{},
	formErrors map[string][]core.ValidationError,
) error {
	if ctx.renderer == nil {
		panic("Cannot call render. Missing dependency: phx.Renderer")
	}
	return ctx.renderer.RenderWithErrors(ctx, status, parsed, vm, formErrors)
}

type CookieOptions struct {
	Name    string
	Expires time.Duration
	Value   string

	// Set if front scripts can access to cookie.
	// If its true, front script cannot access.
	// By default true.
	HttpOnly bool

	// Sets if cookies are only send through https.
	// If its true means https only. By default false.
	Secure bool
}

func (ctx *Context) CreateCookieOptions(opt CookieOptions) error {
	var data string
	if ctx.cy != nil {
		var err error
		data, err = cypher.EncodeCookie(ctx.cy, opt.Value)
		if err != nil {
			return fmt.Errorf("error encoding cookie: %w", err)
		}
	} else {
		log.Println("[PHX] WARNING!: Using plain cookies. " +
			"You must provide a core.Cypher implementation to use encoded cookies")
		data = opt.Value
	}
	http.SetCookie(ctx.Res, &http.Cookie{
		Name:     opt.Name,
		Value:    data,
		Expires:  time.Now().Add(opt.Expires),
		MaxAge:   int(opt.Expires.Seconds()),
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
		HttpOnly: opt.HttpOnly,
		Secure:   opt.Secure,
	})
	return nil
}

func (ctx *Context) CreateCookie(name, data string) error {
	return ctx.CreateCookieOptions(CookieOptions{
		Name:     name,
		Expires:  core.OneDayDuration,
		Value:    data,
		HttpOnly: true,
		Secure:   false,
	})
}

func (ctx *Context) ReadCookie(name string) (string, error) {
	cookie, err := ctx.Req.Cookie(name)
	if err != nil {
		return "", fmt.Errorf("error while reading cookie with key: '%s': %w", name, err)
	}
	var data string
	if ctx.cy != nil {
		data, err = cypher.DecodeCookie(ctx.cy, cookie.Value)
		if err != nil {
			return "", fmt.Errorf("cannot decode cookie: %w", err)
		}
	} else {
		log.Println("[PHX] WARNING!: Using plain cookies. " +
			"You must provide a core.Cypher implementation to use encoded cookies")
		data = cookie.Value
	}
	return data, nil
}
func (ctx *Context) DeleteCookie(name string) error {
	if ctx.cy == nil {
		log.Println("[PHX] WARNING!: Using plain cookies. " +
			"You must provide a core.Cypher implementation to use encoded cookies")
	}
	_, err := ctx.Req.Cookie(name)
	if err != nil {
		return fmt.Errorf("error while reading cookie with key: '%s': %w", name, err)
	}
	http.SetCookie(ctx.Res, &http.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   0,
		Path:     "/",
		SameSite: http.SameSiteDefaultMode,
	})
	return nil
}
