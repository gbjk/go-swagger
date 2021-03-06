package middleware

import (
	"net/http"
	"reflect"

	"github.com/go-swagger/go-swagger/errors"
	"github.com/go-swagger/go-swagger/httpkit"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/strfmt"
)

// RequestBinder binds and validates the data from a http request
type untypedRequestBinder struct {
	Spec         *spec.Swagger
	Parameters   map[string]spec.Parameter
	Formats      strfmt.Registry
	paramBinders map[string]*untypedParamBinder
}

// NewRequestBinder creates a new binder for reading a request.
func newUntypedRequestBinder(parameters map[string]spec.Parameter, spec *spec.Swagger, formats strfmt.Registry) *untypedRequestBinder {
	binders := make(map[string]*untypedParamBinder)
	for fieldName, param := range parameters {
		binders[fieldName] = newUntypedParamBinder(param, spec, formats)
	}
	return &untypedRequestBinder{
		Parameters:   parameters,
		paramBinders: binders,
		Spec:         spec,
		Formats:      formats,
	}
}

// Bind perform the databinding and validation
func (o *untypedRequestBinder) Bind(request *http.Request, routeParams RouteParams, consumer httpkit.Consumer, data interface{}) error {
	val := reflect.Indirect(reflect.ValueOf(data))
	isMap := val.Kind() == reflect.Map
	var result []error

	for fieldName, param := range o.Parameters {
		binder := o.paramBinders[fieldName]

		var target reflect.Value
		if !isMap {
			binder.Name = fieldName
			target = val.FieldByName(fieldName)
		}

		if isMap {
			tpe := binder.Type()
			if tpe == nil {
				if param.Schema.Type.Contains("array") {
					tpe = reflect.TypeOf([]interface{}{})
				} else {
					tpe = reflect.TypeOf(map[string]interface{}{})
				}
			}
			target = reflect.Indirect(reflect.New(tpe))

		}

		if !target.IsValid() {
			result = append(result, errors.New(500, "parameter name %q is an unknown field", binder.Name))
			continue
		}

		if err := binder.Bind(request, routeParams, consumer, target); err != nil {
			result = append(result, err)
			continue
		}

		if binder.validator != nil {
			rr := binder.validator.Validate(target.Interface())
			if rr != nil && rr.HasErrors() {
				result = append(result, rr.AsError())
			}
		}

		if isMap {
			val.SetMapIndex(reflect.ValueOf(param.Name), target)
		}
	}

	if len(result) > 0 {
		return errors.CompositeValidationError(result...)
	}

	return nil
}
