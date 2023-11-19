package phx

import (
	"reflect"
	"strconv"
)

// ParseForm parses req.Form and then serializes the form data to
// the dst struct using reflection. The form names should match to
// the 'html' tag or, if its not setted, to the field name.
// ParseForm only supports serializing data to one-depth structs.
// Example, using
// this struct as target:
//
//	type MyStruct struct {
//			A bool,
//			B int `html:"pagination.currentPage"`
//	}
//
// And having this serialized form:
//
// "A=false&pagination.currentPage=2"
//
// Calling ParseForm like this (NOTE THAT A POINTER TO THE STRUCT IS BEING PASSED):
//
// var s MyStruct
// phx.ParseForm(req, &s)
//
// It will result to this fullfilled struct:
//
// { A = false, B: 2 }
//
// The supported field types are: int, int8, int16, int32, int64, float32, float64, bool and string
func (ctx *Context) ParseForm(dst interface{}) {
	ctx.Req.ParseForm()
	v := reflect.ValueOf(dst)
	// Is a pointer to an interface. An interface is a pointer to something else.
	e := v.Elem()
	t := e.Type()
	if t.Kind() != reflect.Struct {
		return
	}
	num := t.NumField()
	for i := 0; i < num; i++ {
		fieldValue := e.Field(i)
		fieldType := t.Field(i)
		var lookup string
		lookup, ok := fieldType.Tag.Lookup("html")
		if !ok {
			lookup = fieldType.Name
		}
		if !ctx.Req.Form.Has(lookup) {
			continue
		}
		if !fieldValue.IsValid() {
			continue
		}
		if !fieldValue.CanSet() {
			continue
		}
		value := ctx.Req.Form.Get(lookup)
		setValue(fieldValue, value)
	}
}

func setValue(field reflect.Value, value string) bool {
	t := field.Type()
	isPointer := false
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		isPointer = true
	}
	switch t.Kind() {
	case reflect.String:
		return setString(field, value, isPointer)
	case reflect.Int:
		return setInt[int](field, value, isPointer, 64)
	case reflect.Int64:
		return setInt[int64](field, value, isPointer, 64)
	case reflect.Int32:
		return setInt[int32](field, value, isPointer, 32)
	case reflect.Int16:
		return setInt[int16](field, value, isPointer, 16)
	case reflect.Int8:
		return setInt[int8](field, value, isPointer, 8)
	case reflect.Float64:
		return setFloat[float64](field, value, isPointer, 64)
	case reflect.Float32:
		return setFloat[float32](field, value, isPointer, 32)
	case reflect.Bool:
		return setBool(field, value, isPointer)
	default:
		return false
	}
}

func setInt[T int | int8 | int16 | int32 | int64](field reflect.Value, value string, isPointer bool, bits int) bool {
	i, err := strconv.ParseInt(value, 0, bits)
	if err != nil {
		return false
	}
	p := T(i)
	if isPointer {
		if value == "" {
			return true
		}
		field.Set(reflect.ValueOf(&p))
	} else {
		field.Set(reflect.ValueOf(p))
	}
	return true
}

func setFloat[T float64 | float32](field reflect.Value, value string, isPointer bool, bits int) bool {
	f, err := strconv.ParseFloat(value, bits)
	if err != nil {
		return false
	}
	p := T(f)
	if isPointer {
		if value == "" {
			return true
		}
		field.Set(reflect.ValueOf(&p))
	} else {
		field.Set(reflect.ValueOf(p))
	}
	return true
}

func setBool(field reflect.Value, value string, isPointer bool) bool {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	if isPointer {
		if value == "" {
			return true
		}
		field.Set(reflect.ValueOf(&b))
	} else {
		field.Set(reflect.ValueOf(b))
	}
	field.Set(reflect.ValueOf(b))
	return true
}

func setString(field reflect.Value, value string, isPointer bool) bool {
	if isPointer {
		if value == "" {
			return true
		}
		field.Set(reflect.ValueOf(&value))
	} else {
		field.Set(reflect.ValueOf(value))
	}
	return true
}
