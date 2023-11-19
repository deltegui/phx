package phx

import (
	"log"
	"reflect"
)

// Builder is a function that expects anything and retuns
// the type that builds. The type cant be func() interface{}
// cause some errors appears in runtime. So it's represented
// as an interface.
type Builder interface{}

// Injector is an automated dependency injector inspired in Sping's
// DI. It will detect which builder to call using its return type.
// If the builder haver params, it will fullfill that params calling
// other builders that provides its types.
type Injector struct {
	builders map[reflect.Type]Builder
}

// NewInjector with default values
func NewInjector() *Injector {
	return &Injector{
		builders: make(map[reflect.Type]Builder),
	}
}

// Add a builder to the dependency injector.
func (injector Injector) Add(builder Builder) {
	outputType := reflect.TypeOf(builder).Out(0)
	injector.builders[outputType] = builder
}

// ShowAvailableBuilders prints all registered builders.
func (injector Injector) ShowAvailableBuilders() {
	for k := range injector.builders {
		log.Printf("Builder for type: %s\n", k)
	}
}

// Get returns a builded dependency
func (injector Injector) Get(name interface{}) interface{} {
	return injector.GetByType(reflect.TypeOf(name))
}

// GetByType returns a builded dependency identified by type
func (injector Injector) GetByType(name reflect.Type) interface{} {
	dependencyBuilder := injector.builders[name]
	if dependencyBuilder == nil {
		log.Panicf("Builder not found for type %s\n", name)
	}
	return injector.CallBuilder(dependencyBuilder)
}

// ResolveHandler created by a builder
func (injector Injector) ResolveHandler(builder Builder) Handler {
	return injector.CallBuilder(builder).(Handler)
}

// CallBuilder injecting all parameters with provided builders. If some parameter
// type cannot be found, it will panic
func (injector Injector) CallBuilder(builder Builder) interface{} {
	var inputs []reflect.Value
	builderType := reflect.TypeOf(builder)
	for i := 0; i < builderType.NumIn(); i++ {
		impl := injector.GetByType(builderType.In(i))
		inputs = append(inputs, reflect.ValueOf(impl))
	}
	builderVal := reflect.ValueOf(builder)
	builded := builderVal.Call(inputs)
	return builded[0].Interface()
}

// PopulateStruct fills a struct with the implementations
// that the injector can create. Make sure you pass a reference and
// not a value
func (injector Injector) PopulateStruct(userStruct interface{}) {
	ptrStructValue := reflect.ValueOf(userStruct)
	structValue := ptrStructValue.Elem()
	if structValue.Kind() != reflect.Struct {
		log.Panicln("Value passed to PopulateStruct is not a struct")
	}
	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		if field.IsValid() && field.CanSet() {
			impl := injector.GetByType(field.Type())
			field.Set(reflect.ValueOf(impl))
		}
	}
}
