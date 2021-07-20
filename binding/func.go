package binding

import (
	jsonpkg "encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/henrylee2cn/ameda"
	"google.golang.org/protobuf/proto"
)

// JSONUnmarshaler is the interface implemented by types
// that can unmarshal a JSON description of themselves.
type JSONUnmarshaler func(data []byte, v interface{}) error

var (
	jsonUnmarshalFunc func(data []byte, v interface{}) error
)

// ResetJSONUnmarshaler reset the JSON Unmarshal function.
// NOTE: verifyingRequired is true if the required tag is supported.
func ResetJSONUnmarshaler(fn JSONUnmarshaler) {
	jsonUnmarshalFunc = fn
}

var typeUnmarshalFuncs = make(map[reflect.Type]func(string, bool) (reflect.Value, error))

func unsafeUnmarshalValue(v reflect.Value, s string, looseZeroMode bool) error {
	fn := typeUnmarshalFuncs[v.Type()]
	if fn != nil {
		vv, err := fn(s, looseZeroMode)
		if err == nil {
			v.Set(vv)
		}
		return err
	}
	return unmarshal(ameda.UnsafeStringToBytes(s), v.Addr().Interface())
}

func unsafeUnmarshalSlice(t reflect.Type, a []string, looseZeroMode bool) (reflect.Value, error) {
	var err error
	fn := typeUnmarshalFuncs[t]
	if fn == nil {
		fn = func(s string, _ bool) (reflect.Value, error) {
			v := reflect.New(t)
			i := v.Interface()
			err = unmarshal(ameda.UnsafeStringToBytes(s), i)
			return v.Elem(), err
		}
	}
	v := reflect.New(reflect.SliceOf(t)).Elem()
	for _, s := range a {
		var vv reflect.Value
		vv, err = fn(s, looseZeroMode)
		if err != nil {
			return v, err
		}
		v = reflect.Append(v, vv)
	}
	return v, nil
}

func unmarshal(b []byte, i interface{}) error {
	switch x := i.(type) {
	case jsonpkg.Unmarshaler:
		return x.UnmarshalJSON(b)
	case proto.Message:
		return proto.Unmarshal(b, x)
	default:
		return jsonpkg.Unmarshal(b, i)
	}
}

// MustRegTypeUnmarshal registers unmarshalor function of type.
// NOTE:
//  panic if exist error.
func MustRegTypeUnmarshal(t reflect.Type, fn func(v string, emptyAsZero bool) (reflect.Value, error)) {
	err := RegTypeUnmarshal(t, fn)
	if err != nil {
		panic(err)
	}
}

// RegTypeUnmarshal registers unmarshalor function of type.
func RegTypeUnmarshal(t reflect.Type, fn func(v string, emptyAsZero bool) (reflect.Value, error)) error {
	// check
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8,
		reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return errors.New("registration type cannot be a basic type")
	case reflect.Ptr:
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
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(t), nil
	})
}
