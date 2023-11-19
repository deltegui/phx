package validator

import (
	"reflect"

	"github.com/deltegui/phx/core"
	playgroundValidator "github.com/go-playground/validator/v10"
)

func ModelError(errs []ValidationError) map[string]string {
	vmErr := map[string]string{}
	for _, e := range errs {
		vmErr[e.Field] = e.Tag
	}
	return vmErr
}

type ValidationError struct {
	// Tag is the condition that have failed
	Tag string

	// Complete path to the field that have the error.
	Path string

	// Field is the name (and only the name) of the failing field
	Field string

	// Error is the stringified error
	Err string

	Value interface{}
	Kind  reflect.Kind
}

func (v ValidationError) Error() string {
	return v.Err
}

type PlaygroundValidator struct {
	validator *playgroundValidator.Validate
}

func NewPlayground() PlaygroundValidator {
	return PlaygroundValidator{validator: playgroundValidator.New()}
}

func New() core.Validator {
	val := NewPlayground()
	return func(t interface{}) map[string]string {
		ss, err := val.Validate(t)
		if err != nil {
			panic(err)
		}
		if len(ss) == 0 {
			return nil
		}
		return ModelError(ss)
	}
}

func (val PlaygroundValidator) Validate(target interface{}) ([]ValidationError, error) {
	err := val.validator.Struct(target)
	if err != nil {
		e, ok := err.(playgroundValidator.ValidationErrors)
		if !ok {
			return nil, err
		}
		return errorsToResult(e), nil
	}
	return []ValidationError{}, nil
}

func errorsToResult(ee playgroundValidator.ValidationErrors) []ValidationError {
	result := make([]ValidationError, len(ee))
	for i, e := range ee {
		result[i] = ValidationError{
			Tag:   e.ActualTag(),
			Path:  e.StructNamespace(),
			Field: e.Field(),
			Err:   e.Error(),
			Value: e.Value(),
			Kind:  e.Kind(),
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
