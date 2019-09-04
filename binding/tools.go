package binding

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/henrylee2cn/goutil"
)

func copyBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	b, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(b))
	return b, nil
}

func getParamName(eval func() interface{}, defaultName string) (name string, errStr string) {
	name, errStr = evalString(eval)
	if errStr == "" || name != "" {
		return
	}
	name = defaultName
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

var errMismatch = errors.New("type mismatch")

func stringsToValue(t reflect.Type, a []string, emptyAsZero bool) (reflect.Value, error) {
	var i interface{}
	var err error
	var ptrDepth int
	elmeKind := t.Kind()
	for elmeKind == reflect.Ptr {
		t = t.Elem()
		elmeKind = t.Kind()
		ptrDepth++
	}
	switch elmeKind {
	case reflect.String:
		i = a
	case reflect.Bool:
		i, err = goutil.StringsToBools(a, emptyAsZero)
	case reflect.Float32:
		i, err = goutil.StringsToFloat32s(a, emptyAsZero)
	case reflect.Float64:
		i, err = goutil.StringsToFloat64s(a, emptyAsZero)
	case reflect.Int:
		i, err = goutil.StringsToInts(a, emptyAsZero)
	case reflect.Int64:
		i, err = goutil.StringsToInt64s(a, emptyAsZero)
	case reflect.Int32:
		i, err = goutil.StringsToInt32s(a, emptyAsZero)
	case reflect.Int16:
		i, err = goutil.StringsToInt16s(a, emptyAsZero)
	case reflect.Int8:
		i, err = goutil.StringsToInt8s(a, emptyAsZero)
	case reflect.Uint:
		i, err = goutil.StringsToUints(a, emptyAsZero)
	case reflect.Uint64:
		i, err = goutil.StringsToUint64s(a, emptyAsZero)
	case reflect.Uint32:
		i, err = goutil.StringsToUint32s(a, emptyAsZero)
	case reflect.Uint16:
		i, err = goutil.StringsToUint16s(a, emptyAsZero)
	case reflect.Uint8:
		i, err = goutil.StringsToUint8s(a, emptyAsZero)
	default:
		fn := typeUnmarshalFuncs[t]
		if fn == nil {
			return reflect.Value{}, errMismatch
		}
		v := reflect.New(reflect.SliceOf(t)).Elem()
		for _, s := range a {
			vv, err := fn(s, emptyAsZero)
			if err != nil {
				return reflect.Value{}, errMismatch
			}
			v = reflect.Append(v, vv)
		}
		return goutil.ReferenceSlice(v, ptrDepth), nil
	}
	if err != nil {
		return reflect.Value{}, errMismatch
	}
	return goutil.ReferenceSlice(reflect.ValueOf(i), ptrDepth), nil
}
