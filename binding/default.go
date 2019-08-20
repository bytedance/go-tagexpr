package binding

import "net/http"

var defaultBinding = New(nil)

// Default returns the default binding.
// NOTE:
//  path tag name is 'path';
//  query tag name is 'query';
//  header tag name is 'header';
//  cookie tag name is 'cookie';
//  raw_body tag name is 'raw_body';
//  form tag name is 'form';
//  validator tag name is 'vd';
//  protobuf tag name is 'protobuf';
//  json tag name is 'json';
//  LooseZeroMode is false.
func Default() *Binding {
	return defaultBinding
}

// SetLooseZeroMode if set to true,
// the empty string request parameter is bound to the zero value of parameter.
// NOTE:
//  The default is false;
//  Suitable for these parameter types: query/header/cookie/form .
func SetLooseZeroMode(enable bool) {
	defaultBinding.SetLooseZeroMode(enable)
}

// SetErrorFactory customizes the factory of validation error.
// NOTE:
//  If errFactory==nil, the default is used
func SetErrorFactory(bindErrFactory, validatingErrFactory func(failField, msg string) error) {
	defaultBinding.SetErrorFactory(bindErrFactory, validatingErrFactory)
}

// BindAndValidate binds the request parameters and validates them if needed.
func BindAndValidate(structPointer interface{}, req *http.Request, pathParams PathParams) error {
	return defaultBinding.BindAndValidate(structPointer, req, pathParams)
}

// Bind binds the request parameters.
func Bind(structPointer interface{}, req *http.Request, pathParams PathParams) error {
	return defaultBinding.Bind(structPointer, req, pathParams)
}

// Validate validates whether the fields of value is valid.
func Validate(value interface{}) error {
	return defaultBinding.Validate(value)
}
