package phx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/pagination"
	"github.com/deltegui/phx/session"
	"github.com/julienschmidt/httprouter"
)

type Context struct {
	Req    *http.Request
	Res    http.ResponseWriter
	params httprouter.Params
	ctx    context.Context

	locstore *localizer.LocalizerStore

	renderer Renderer

	validate core.Validator
}

func (r *Context) Set(key, value any) {
	r.ctx = context.WithValue(r.ctx, key, value)
}

func (r *Context) Get(key any) any {
	return r.ctx.Value(key)
}

func (ctx *Context) PaginationToVM(pag pagination.Pagination) pagination.ViewModel {
	return pagination.PaginationToVM(pag, ctx.GetLocalizer("common/pagination"))
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
		log.Println("Call to HaveSession. You dont have sessions enabled. To do so, you have to add a middleware.")
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

func (ctx *Context) GetUrlParam(name string) string {
	return ctx.params.ByName(name)
}

func (ctx *Context) GetQueryParam(name string) string {
	return ctx.Req.URL.Query().Get(name)
}

func (ctx *Context) GetCurrentLanguage() string {
	return ctx.locstore.ReadCookie(ctx.Req)
}

func (ctx *Context) ChangeLanguage(to string) {
	ctx.locstore.CreateCookie(ctx.Res, to)
}

func (ctx *Context) Validate(s any) map[string]string {
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

func (ctx *Context) OK(data string, a ...any) error {
	return ctx.String(http.StatusOK, data, a...)
}

func (ctx *Context) InternalServerError(data string, a ...any) error {
	return ctx.String(http.StatusInternalServerError, data, a...)
}

func (ctx *Context) Json(status int, data any) error {
	response, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %s", err)
	}
	ctx.Res.WriteHeader(status)
	ctx.Res.Header().Set("Content-Type", "application/json")
	ctx.Res.Write(response)
	return nil
}

func (ctx *Context) JsonOK(data interface{}) error {
	return ctx.Json(http.StatusOK, data)
}

func (ctx *Context) Render(status int, parsed string, vm interface{}) error {
	if ctx.renderer == nil {
		panic("Cannot call render. Missing dependency: phx.Renderer")
	}
	return ctx.renderer.Render(ctx, status, parsed, vm)
}

func (ctx *Context) RenderOK(parsed string, vm interface{}) error {
	return ctx.Render(http.StatusOK, parsed, vm)
}

func (ctx *Context) RenderWithErrors(status int, parsed string, vm interface{}, formErrors map[string]string) error {
	if ctx.renderer == nil {
		panic("Cannot call render. Missing dependency: phx.Renderer")
	}
	return ctx.renderer.RenderWithErrors(ctx, status, parsed, vm, formErrors)
}
