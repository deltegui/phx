package renderer

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/localizer"
	"github.com/deltegui/phx/model"
)

type TemplateRenderer struct {
	tmpl      map[string]*template.Template
	tmplFuncs template.FuncMap
	tmplFS    embed.FS
}

func NewTemplateRenderer(fs embed.FS) *TemplateRenderer {
	return &TemplateRenderer{
		tmpl:      make(map[string]*template.Template),
		tmplFS:    fs,
		tmplFuncs: map[string]any{},
	}
}

func (r *TemplateRenderer) ShowAvailableTemplates() {
	log.Println("Templates:")
	for key, value := range r.tmpl {
		log.Println(key, "->", value.Tree.Name)
	}
}

func (r *TemplateRenderer) Render(ctx *phx.Context, status int, parsed string, vm interface{}) error {
	if ctx == nil {
		panic("Called to Render outside request: no context!")
	}
	ctx.Res.WriteHeader(status)
	model := model.CreateViewModel(ctx, parsed, vm)
	template, ok := r.tmpl[parsed]
	if !ok {
		return fmt.Errorf("error executing template with parsed name: '%s'. It does not exists", parsed)
	}
	if err := template.Execute(ctx.Res, model); err != nil {
		return fmt.Errorf("error executing tempalte with parsed name '%s': %s", parsed, err)
	}
	return nil
}

func (r *TemplateRenderer) RenderWithErrors(ctx *phx.Context, status int, parsed string, vm interface{}, formErrors map[string]string) error {
	if ctx == nil {
		panic("Called to Render outside request: no context!")
	}
	ctx.Res.WriteHeader(status)
	model := model.CreateViewModel(ctx, parsed, vm)
	model.FormErrors = formErrors
	if err := r.tmpl[parsed].Execute(ctx.Res, model); err != nil {
		return fmt.Errorf("error executing tempalte with parsed name '%s': %s", parsed, err)
	}
	return nil
}

func (r *TemplateRenderer) AddDefaultTemplateFunctions() {
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
		"SelectList": func(loc localizer.Localizer, list model.SelectList) model.ViewModel {
			return model.ViewModel{
				Localizer: loc,
				Model:     list,
			}
		},
		"CreateSelectList": func(loc localizer.Localizer, name string, items []model.SelectItem) model.ViewModel {
			return model.CreateSelectListViewModel(loc, name, items, false)
		},
		"CreateMultipleSelectList": func(loc localizer.Localizer, name string, items []model.SelectItem) model.ViewModel {
			return model.CreateSelectListViewModel(loc, name, items, true)
		},
		"YesNoSelectList": model.CreateYesNoSelectListViewModel,
	}
}

// AddTemplateFunction registers a new template function. If you want to have
// default template functions you must call AddDefaultTemplateFunctions before
// call this funciton
func (r *TemplateRenderer) AddTemplateFunction(name string, f any) {
	if r.tmplFuncs == nil {
		r.tmplFuncs = map[string]any{}
	}
	r.tmplFuncs[name] = f
}

func (r *TemplateRenderer) Parse(name, main string, patterns ...string) {
	tmpl := template.New(main)
	if r.tmplFuncs != nil {
		tmpl = tmpl.Funcs(r.tmplFuncs)
	}
	compilation := template.Must(tmpl.ParseFS(r.tmplFS, patterns...))
	r.tmpl[name] = compilation
}

func (r *TemplateRenderer) ParsePartial(name string, patterns ...string) {
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
