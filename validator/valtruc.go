package validator

import (
	"errors"

	"github.com/deltegui/phx/core"
	"github.com/deltegui/valtruc"
)

type valtrucValidationError struct {
	err valtruc.ValidationError
}

func (verr valtrucValidationError) Format(f string) string {
	return verr.err.Format(f)
}

func (verr valtrucValidationError) Error() string {
	return verr.err.Error()
}

func (verr valtrucValidationError) GetName() string {
	return string(verr.err.GetIdentifier())
}

func New() core.Validator {
	vt := valtruc.New()
	return func(i interface{}) map[string][]core.ValidationError {
		errs := vt.Validate(i)
		output := map[string][]core.ValidationError{}

		verr := valtruc.ValidationError{}
		for _, err := range errs {
			ok := errors.As(err, &verr)
			if !ok {
				continue
			}
			fieldName := verr.GetFieldName()
			output[fieldName] = append(output[fieldName], valtrucValidationError{verr})
		}

		return output
	}
}
