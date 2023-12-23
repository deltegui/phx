package model

import (
	"fmt"

	"github.com/deltegui/phx"
	"github.com/deltegui/phx/localizer"
)

type ViewModel struct {
	Model      interface{}
	Localizer  localizer.Localizer
	FormErrors map[string]string
	CsrfToken  string
	Ctx        *phx.Context
}

func CreateViewModel(ctx *phx.Context, name string, model interface{}) ViewModel {
	var loc localizer.Localizer = nil
	/*if ctx.locstore != nil {
		loc = ctx.locstore.GetUsingRequest(name, ctx.Req)
	}*/
	csrfToken, ok := ctx.Get("phx-csrf").(string)
	if !ok {
		csrfToken = ""
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

func CreateSelectListViewModel(loc localizer.Localizer, name string, items []SelectItem, multiple bool) ViewModel {
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

func CreateYesNoSelectListViewModel(loc localizer.Localizer, name string, value *bool) ViewModel {
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
