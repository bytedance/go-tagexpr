package binding

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	_ "unsafe"

	"github.com/bytedance/go-tagexpr"
)

const (
	unsupportBody = 0
	jsonBody      = 1
	formBody      = 1
)

func getBodyCodec(req *http.Request) uint8 {
	switch req.Header.Get("Content-Type") {
	case "application/json":
		return jsonBody
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return formBody
	default:
		return unsupportBody
	}
}

func copyBody(req *http.Request) ([]byte, error) {
	b, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(b))
	return b, nil
}

func getParamName(es tagexpr.ExprSelector, eval func() interface{}) (name string, errStr string) {
	name, errStr = evalString(eval)
	if errStr != "" {
		return
	}
	if name == "" {
		name = es.Field()
	}
	return
}

func evalString(eval func() interface{}) (val string, errStr string) {
	switch r := eval().(type) {
	case string:
		return r, ""
	case nil:
		return "", ""
	default:
		return "", "parameter position value must be a string type"
	}
}

//go:linkname derefType validator.derefType
func derefType(t reflect.Type) reflect.Type

//go:linkname derefValue validator.derefValue
func derefValue(v reflect.Value) reflect.Value

// func derefInitValue(v reflect.Value) reflect.Value {
// 	for v.Kind() == reflect.Ptr {
// 		e := v.Elem()
// 		if e.IsValid() {
// 			e = reflect.New(e.Type())
// 			v.Set(e)
// 		}
// 		v = e
// 	}
// 	return v
// }

// func safeIsNil(v reflect.Value) bool {
// 	if !v.IsValid() {
// 		return true
// 	}
// 	switch v.Kind() {
// 	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr,
// 		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
// 		return v.IsNil()
// 	}
// 	return false
// }

func stringsToInts(a []string) ([]int, error) {
	r := make([]int, len(a))
	for k, v := range a {
		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		r[k] = i
	}
	return r, nil
}

func stringsToInt64s(a []string) ([]int64, error) {
	r := make([]int64, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		r[k] = i
	}
	return r, nil
}

func stringsToInt32s(a []string) ([]int32, error) {
	r := make([]int32, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, err
		}
		r[k] = int32(i)
	}
	return r, nil
}

func stringsToInt16s(a []string) ([]int16, error) {
	r := make([]int16, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return nil, err
		}
		r[k] = int16(i)
	}
	return r, nil
}

func stringsToInt8s(a []string) ([]int8, error) {
	r := make([]int8, len(a))
	for k, v := range a {
		i, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return nil, err
		}
		r[k] = int8(i)
	}
	return r, nil
}

func stringsToUint8s(a []string) ([]uint8, error) {
	r := make([]uint8, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return nil, err
		}
		r[k] = uint8(i)
	}
	return r, nil
}

func stringsToUint16s(a []string) ([]uint16, error) {
	r := make([]uint16, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return nil, err
		}
		r[k] = uint16(i)
	}
	return r, nil
}

func stringsToUint32s(a []string) ([]uint32, error) {
	r := make([]uint32, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, err
		}
		r[k] = uint32(i)
	}
	return r, nil
}

func stringsToUint64s(a []string) ([]uint64, error) {
	r := make([]uint64, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		r[k] = uint64(i)
	}
	return r, nil
}

func stringsToUints(a []string) ([]uint, error) {
	r := make([]uint, len(a))
	for k, v := range a {
		i, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		r[k] = uint(i)
	}
	return r, nil
}

var errMismatch = errors.New("type mismatch")

func stringsToValue(elmeKind reflect.Kind, a []string) (reflect.Value, error) {
	var i interface{}
	var err error
	switch elmeKind {
	case reflect.String:
		i = a
	case reflect.Int:
		i, err = stringsToInts(a)
	case reflect.Int64:
		i, err = stringsToInt64s(a)
	case reflect.Int32:
		i, err = stringsToInt32s(a)
	case reflect.Int16:
		i, err = stringsToInt16s(a)
	case reflect.Int8:
		i, err = stringsToInt8s(a)
	case reflect.Uint:
		i, err = stringsToUints(a)
	case reflect.Uint64:
		i, err = stringsToUint64s(a)
	case reflect.Uint32:
		i, err = stringsToUint32s(a)
	case reflect.Uint16:
		i, err = stringsToUint16s(a)
	case reflect.Uint8:
		i, err = stringsToUint8s(a)
	default:
		return reflect.Value{}, errMismatch
	}
	if err != nil {
		return reflect.Value{}, errMismatch
	}
	return reflect.ValueOf(i), nil
}
