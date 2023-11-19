package phx

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/deltegui/phx/localizer"
)

func (ctx *Context) Json(status int, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		ctx.Res.WriteHeader(http.StatusInternalServerError)
		log.Println("[PHX] Error marshaling data: ", err)
		return
	}
	ctx.Res.WriteHeader(http.StatusBadRequest)
	ctx.Res.Header().Set("Content-Type", "application/json")
	ctx.Res.Write(response)
}

func (ctx *Context) Render(status int, parsed string, vm interface{}) {
	ctx.Res.WriteHeader(status)
	model := ctx.createViewModel(parsed, vm)
	if err := ctx.tmpl[parsed].Execute(ctx.Res, model); err != nil {
		log.Println("[PHX] Error executing tempalte with parsed name '%s': %s", parsed, err)
	}
}

func (ctx *Context) RenderWithErrors(status int, parsed string, vm interface{}, formErrors map[string]string) {
	ctx.Res.WriteHeader(status)
	model := ctx.createViewModel(parsed, vm)
	model.FormErrors = formErrors
	if err := ctx.tmpl[parsed].Execute(ctx.Res, model); err != nil {
		log.Println("[PHX] Error executing tempalte with parsed name '%s': %s", parsed, err)
	}
}

func (r *Router) AddDefaultTemplateFunctions() {
	r.tmplFuncs = map[string]any{
		"Uppercase": func(v string) string {
			return strings.ToUpper(v)
		},
		"StringNotEmpty": func(v string) bool {
			return len(v) > 0
		},
		"BoolToYesNo": func(b bool) string {
			if b {
				//return loc.Get("shared.yes")
				return "Sí"
			} else {
				//return loc.Get("shared.no")
				return "No"
			}
		},
	}
}

func (r *Router) Parse(name string, patterns ...string) {
	tmpl := template.New(name)
	if r.tmplFuncs != nil {
		tmpl = tmpl.Funcs(r.tmplFuncs)
	}
	compilation := template.Must(tmpl.ParseFS(r.tmplFS, patterns...))
	r.tmpl[name] = compilation
}

func (r *Router) ParsePartial(name string, patterns ...string) {
	tmpl := template.New("partial")
	if r.tmplFuncs != nil {
		tmpl.Funcs(r.tmplFuncs)
	}
	compilation, err := tmpl.ParseFS(r.tmplFS, patterns...)
	if err != nil {
		log.Panicln("Failed to parse partial view:", err)
	}
	main := fmt.Sprintf("{{ template \"%s\" . }}", name)
	r.tmpl[name] = template.Must(compilation.Parse(main))
}

type ViewModel struct {
	Model      interface{}
	Localizer  localizer.Localizer
	FormErrors map[string]string
	CsrfToken  string
}

func (ctx *Context) createViewModel(name string, model interface{}) ViewModel {
	var loc localizer.Localizer = nil
	if ctx.locstore != nil {
		loc = ctx.locstore.GetUsingRequest(name, ctx.Req)
	}
	return ViewModel{
		Model:     model,
		CsrfToken: ctx.csrf.Generate(),
		Localizer: loc,
	}
}

func (vm ViewModel) Localize(key string) string {
	return vm.Localizer.Get(key)
}

func (vm ViewModel) HaveFormError(key string) bool {
	if vm.FormErrors == nil {
		return false
	}
	_, ok := vm.FormErrors[key]
	return ok
}

func (vm ViewModel) GetFormError(key string) string {
	if vm.FormErrors == nil {
		return ""
	}
	val, ok := vm.FormErrors[key]
	if !ok {
		return ""
	}
	locVal := vm.Localize(val)
	locKey := vm.Localize(key)
	return fmt.Sprintf(locVal, locKey)
}
