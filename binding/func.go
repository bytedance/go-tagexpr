package binding

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

var (
	jsonUnmarshalFunc       func(data []byte, v interface{}) error
	jsonIndependentRequired = true
)

// ResetJSONUnmarshaler reset the JSON Unmarshal function.
// NOTE: verifyingRequired is true if the required tag is supported.
func ResetJSONUnmarshaler(verifyingRequired bool, fn func(data []byte, v interface{}) error) {
	jsonIndependentRequired = !verifyingRequired
	jsonUnmarshalFunc = fn
}

var typeUnmarshalFuncs = make(map[reflect.Type]func(string, bool) (reflect.Value, error))

// MustRegTypeUnmarshal registers unmarshalor function of type.
// NOTE:
//  panic if return error.
func MustRegTypeUnmarshal(t reflect.Type, fn func(v string, emptyAsZero bool) (reflect.Value, error)) {
	err := RegTypeUnmarshal(t, fn)
	if err != nil {
		panic(err)
	}
}

// RegTypeUnmarshal registers unmarshalor function of type.
func RegTypeUnmarshal(t reflect.Type, fn func(v string, emptyAsZero bool) (reflect.Value, error)) error {
	// check
	if t.Kind() == reflect.Ptr {
		return errors.New("registration type cannot be a pointer type")
	}
	// test
	vv, err := fn("", true)
	if err != nil {
		return fmt.Errorf("test fail: %s", err)
	}
	if tt := vv.Type(); tt != t {
		return fmt.Errorf("test fail: expect return value type is %s, but got %s", t.String(), tt.String())
	}

	typeUnmarshalFuncs[t] = fn
	return nil
}

func init() {
	MustRegTypeUnmarshal(reflect.TypeOf(time.Time{}), func(v string, emptyAsZero bool) (reflect.Value, error) {
		if v == "" && emptyAsZero {
			return reflect.ValueOf(time.Time{}), nil
		}
		t, err := time.Parse(time.RFC1123, v)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(t), nil
	})
}
