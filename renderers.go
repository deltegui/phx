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
	ctx.Res.WriteHeader(status)
	model := ctx.createViewModel(parsed, vm)
	template, ok := ctx.tmpl[parsed]
	if !ok {
		return fmt.Errorf("error executing template with parsed name: '%s'. It does not exists", parsed)
	}
	if err := template.Execute(ctx.Res, model); err != nil {
		return fmt.Errorf("error executing tempalte with parsed name '%s': %s", parsed, err)
	}
	return nil
}

func (ctx *Context) RenderOK(parsed string, vm interface{}) error {
	return ctx.Render(http.StatusOK, parsed, vm)
}

func (ctx *Context) RenderWithErrors(status int, parsed string, vm interface{}, formErrors map[string]string) error {
	ctx.Res.WriteHeader(status)
	model := ctx.createViewModel(parsed, vm)
	model.FormErrors = formErrors
	if err := ctx.tmpl[parsed].Execute(ctx.Res, model); err != nil {
		return fmt.Errorf("error executing tempalte with parsed name '%s': %s", parsed, err)
	}
	return nil
}

type SelectItem struct {
	Value    string
	Tag      string
	Selected bool
}

type SelectList struct {
	Name     string
	Multiple bool
	Items    []SelectItem
}

func createSelectListViewModel(loc localizer.Localizer, name string, items []SelectItem, multiple bool) ViewModel {
	list := SelectList{
		Name:     name,
		Multiple: multiple,
		Items:    items,
	}
	return ViewModel{
		Model:     list,
		Localizer: loc,
	}
}

func createYesNoSelectListViewModel(loc localizer.Localizer, name string, value *bool) ViewModel {
	items := []SelectItem{
		{
			Value:    "",
			Tag:      "shared.choose",
			Selected: value == nil,
		},
		{
			Value:    "1",
			Tag:      "shared.yes",
			Selected: value != nil && *value,
		},
		{
			Value:    "0",
			Tag:      "shared.no",
			Selected: value != nil && !*value,
		},
	}
	list := SelectList{
		Name:     name,
		Multiple: false,
		Items:    items,
	}
	return ViewModel{
		Model:     list,
		Localizer: loc,
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
		"SelectList": func(loc localizer.Localizer, list SelectList) ViewModel {
			return ViewModel{
				Localizer: loc,
				Model:     list,
			}
		},
		"CreateSelectList": func(loc localizer.Localizer, name string, items []SelectItem) ViewModel {
			return createSelectListViewModel(loc, name, items, false)
		},
		"CreateMultipleSelectList": func(loc localizer.Localizer, name string, items []SelectItem) ViewModel {
			return createSelectListViewModel(loc, name, items, true)
		},
		"YesNoSelectList": createYesNoSelectListViewModel,
	}
}

// AddTemplateFunction registers a new template function. If you want to have
// default template functions you must call AddDefaultTemplateFunctions before
// call this funciton
func (r *Router) AddTemplateFunction(name string, f any) {
	if r.tmplFuncs == nil {
		r.tmplFuncs = map[string]any{}
	}
	r.tmplFuncs[name] = f
}

func (r *Router) Parse(name, main string, patterns ...string) {
	tmpl := template.New(main)
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
	Ctx        *Context
}

func (ctx *Context) createViewModel(name string, model interface{}) ViewModel {
	var loc localizer.Localizer = nil
	if ctx.locstore != nil {
		loc = ctx.locstore.GetUsingRequest(name, ctx.Req)
	}
	csrfToken := ""
	if ctx.csrf != nil {
		csrfToken = ctx.csrf.Generate()
	}
	return ViewModel{
		Model:     model,
		CsrfToken: csrfToken,
		Localizer: loc,
		Ctx:       ctx,
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
