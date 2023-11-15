package phx

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/deltegui/phx/csrf"
	"github.com/deltegui/phx/localizer"
)

type Render func(data interface{}, err error)

func NewJsonRenderer(w http.ResponseWriter, req *http.Request) Render {
	render := func(data interface{}) {
		response, err := json.Marshal(data)
		if err != nil {
			log.Println("Error marshaling data: ", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
	return func(data interface{}, errs error) {
		if errs != nil {
			w.WriteHeader(http.StatusBadRequest)
			render(errs)
			return
		}
		render(data)
	}
}

var funcs template.FuncMap = map[string]any{
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

func NewHtmlRenderer(files embed.FS, csrf csrf.Csrf, localizer localizer.LocalizerStore) HtmlRenderer {
	return HtmlRenderer{
		files:     files,
		csrf:      csrf,
		funcs:     funcs,
		localizer: localizer,
	}
}

func NewHtmlRendererWithExtraFuncs(files embed.FS, csrf csrf.Csrf, localizer localizer.LocalizerStore, extra template.FuncMap) HtmlRenderer {
	f := mergeMaps(funcs, extra)
	return HtmlRenderer{
		files:     files,
		csrf:      csrf,
		funcs:     f,
		localizer: localizer,
	}
}

func mergeMaps(m1 map[string]any, m2 map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

type HtmlRenderer struct {
	files     embed.FS
	funcs     template.FuncMap
	csrf      csrf.Csrf
	localizer localizer.LocalizerStore
}

func (r HtmlRenderer) Parse(patterns ...string) *template.Template {
	return template.Must(
		template.
			New("layout.html").
			Funcs(r.funcs).
			ParseFS(r.files, patterns...))
}

func (r HtmlRenderer) ParseWithShared(patterns ...string) *template.Template {
	patterns = append([]string{"shared/*.html"}, patterns...)
	return r.Parse(patterns...)
}

func (r HtmlRenderer) ParseAlone(file string) *template.Template {
	return r.Parse("shared/layout.html", file)
}

func (r HtmlRenderer) ParsePartial(defName string, patterns ...string) *template.Template {
	tmpl, err := template.
		New("partial").
		Funcs(r.funcs).
		ParseFS(r.files, patterns...)
	if err != nil {
		log.Panicln("Failed to parse partial view:", err)
	}
	main := fmt.Sprintf("{{ template \"%s\" . }}", defName)
	return template.Must(tmpl.Parse(main))
}

type View struct {
	template  *template.Template
	model     ViewModel
	localizer localizer.LocalizerStore
}

func (r HtmlRenderer) CreateView(req *http.Request, tmpl *template.Template, model interface{}) View {
	return View{
		template: tmpl,
		model: ViewModel{
			Model:     model,
			CsrfToken: r.csrf.Generate(),
		},
		localizer: r.localizer,
	}
}

func (view *View) FillFormErrors(e map[string]string) {
	view.model.FormErrors = e
}

func (view View) Execute(w io.Writer) error {
	return view.template.Execute(w, view.model)
}

func (view View) RenderLocalized(w io.Writer, req *http.Request, localizerKey string) error {
	view.model.Localizer = view.localizer.GetUsingRequest(localizerKey, req)
	return view.Execute(w)
}

type ViewModel struct {
	Model      interface{}
	Localizer  localizer.Localizer
	FormErrors map[string]string
	CsrfToken  string
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
