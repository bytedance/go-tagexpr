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

func stringsToValue(elmeKind reflect.Kind, a []string) (reflect.Value, error) {
	var i interface{}
	var err error
	switch elmeKind {
	case reflect.String:
		i = a
	case reflect.Bool:
		i, err = goutil.StringsToBools(a)
	case reflect.Float32:
		i, err = goutil.StringsToFloat32s(a)
	case reflect.Float64:
		i, err = goutil.StringsToFloat64s(a)
	case reflect.Int:
		i, err = goutil.StringsToInts(a)
	case reflect.Int64:
		i, err = goutil.StringsToInt64s(a)
	case reflect.Int32:
		i, err = goutil.StringsToInt32s(a)
	case reflect.Int16:
		i, err = goutil.StringsToInt16s(a)
	case reflect.Int8:
		i, err = goutil.StringsToInt8s(a)
	case reflect.Uint:
		i, err = goutil.StringsToUints(a)
	case reflect.Uint64:
		i, err = goutil.StringsToUint64s(a)
	case reflect.Uint32:
		i, err = goutil.StringsToUint32s(a)
	case reflect.Uint16:
		i, err = goutil.StringsToUint16s(a)
	case reflect.Uint8:
		i, err = goutil.StringsToUint8s(a)
	default:
		return reflect.Value{}, errMismatch
	}
	if err != nil {
		return reflect.Value{}, errMismatch
	}
	return reflect.ValueOf(i), nil
}
