package validator_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/deltegui/phx/validator"
)

type loginRequest struct {
	Name     string `validate:"required,min=3,max=255"`
	Password string `validate:"required,min=3,max=255"`
}

func TestShouldCheckAndReturnValidationErrors(t *testing.T) {
	v := validator.NewPlayground()
	t.Run("AllOk", func(t *testing.T) {
		login := loginRequest{
			Name:     "demo",
			Password: "mypass",
		}
		valErr, err := v.Validate(login)
		if err != nil {
			t.Error(err)
		}
		if len(valErr) != 0 {
			t.Error("Expected validation errors to have a length of 0")
		}
	})

	t.Run("Should return errors", func(t *testing.T) {
		login := loginRequest{
			Name:     "d",
			Password: "",
		}
		valErr, err := v.Validate(login)
		if err != nil {
			t.Error(err)
		}
		if len(valErr) != 2 {
			t.Error("Expected to have 2 errors")
		}
		expectedFirst := validator.ValidationError{
			Tag:   "min",
			Path:  "loginRequest.Name",
			Field: "Name",
			Err:   "Key: 'loginRequest.Name' Error:Field validation for 'Name' failed on the 'min' tag",
			Value: "d",
			Kind:  reflect.String,
		}
		if valErr[0] != expectedFirst {
			fmt.Println("Expected:")
			fmt.Println(expectedFirst)
			fmt.Println("But have:")
			fmt.Println(valErr[0])
			t.Error("Errors does not match for first error")
		}
		expectedSecond := validator.ValidationError{
			Tag:   "required",
			Path:  "loginRequest.Password",
			Field: "Password",
			Err:   "Key: 'loginRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag",
			Value: "",
			Kind:  reflect.String,
		}
		if valErr[1] != expectedSecond {
			fmt.Println("Expected:")
			fmt.Println(expectedSecond)
			fmt.Println("But have:")
			fmt.Println(valErr[1])
			t.Error("Errors does not match for second error")
		}
	})
}

func TestValidatorResultMustBeCompatibleWithError(t *testing.T) {
	v := validator.NewPlayground()
	login := loginRequest{
		Name:     "d",
		Password: "",
	}
	valErr, _ := v.Validate(login)
	var errs []validator.ValidationError = valErr // This cast should be Ok
	if len(errs) != len(valErr) {
		t.Error("Expected []error and ValidationResult to have the same length")
	}
}
