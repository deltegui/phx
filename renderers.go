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

func (ctx *Context) String(status int, data string, a ...any) {
	ctx.Res.WriteHeader(status)
	fmt.Fprintf(ctx.Res, data, a...)
}

func (ctx *Context) StringOK(data string, a ...any) {
	ctx.String(http.StatusOK, data, a...)
}

func (ctx *Context) Json(status int, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		ctx.Res.WriteHeader(http.StatusInternalServerError)
		log.Println("[PHX] Error marshaling data: ", err)
		return
	}
	ctx.Res.WriteHeader(status)
	ctx.Res.Header().Set("Content-Type", "application/json")
	ctx.Res.Write(response)
}

func (ctx *Context) JsonOK(data interface{}) {
	ctx.Json(http.StatusOK, data)
}

func (ctx *Context) Render(status int, parsed string, vm interface{}) {
	ctx.Res.WriteHeader(status)
	model := ctx.createViewModel(parsed, vm)
	template, ok := ctx.tmpl[parsed]
	if !ok {
		log.Println("[PHX] Error executing template with parsed name:", parsed, ". It does not exists")
		return
	}
	if err := template.Execute(ctx.Res, model); err != nil {
		log.Printf("[PHX] Error executing tempalte with parsed name '%s': %s\n", parsed, err)
	}
}

func (ctx *Context) RenderOK(parsed string, vm interface{}) {
	ctx.Render(http.StatusOK, parsed, vm)
}

func (ctx *Context) RenderWithErrors(status int, parsed string, vm interface{}, formErrors map[string]string) {
	ctx.Res.WriteHeader(status)
	model := ctx.createViewModel(parsed, vm)
	model.FormErrors = formErrors
	if err := ctx.tmpl[parsed].Execute(ctx.Res, model); err != nil {
		log.Printf("[PHX] Error executing tempalte with parsed name '%s': %s\n", parsed, err)
	}
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
