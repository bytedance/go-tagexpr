package binding

import (
	jsonpkg "encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/goutil"
	"github.com/tidwall/gjson"

	"github.com/bytedance/go-tagexpr"
)

const (
	specialChar = "\x07"
)

type paramInfo struct {
	fieldSelector  string
	structField    reflect.StructField
	tagInfos       []*tagInfo
	omitIns        map[in]bool
	bindErrFactory func(failField, msg string) error
	looseZeroMode  bool
	defaultVal     []byte
}

func (p *paramInfo) name(paramIn in) string {
	var name string
	for _, info := range p.tagInfos {
		if info.paramIn == json {
			name = info.paramName
			break
		}
	}
	if name == "" {
		return p.structField.Name
	}
	return name
}

func (p *paramInfo) getField(expr *tagexpr.TagExpr, initZero bool) (reflect.Value, error) {
	fh, found := expr.Field(p.fieldSelector)
	if found {
		v := fh.Value(initZero)
		if v.IsValid() {
			return v, nil
		}
	}
	return reflect.Value{}, nil
}

func (p *paramInfo) bindRawBody(info *tagInfo, expr *tagexpr.TagExpr, bodyBytes []byte) error {
	if len(bodyBytes) == 0 {
		if info.required {
			return info.requiredError
		}
		return nil
	}
	v, err := p.getField(expr, true)
	if err != nil || !v.IsValid() {
		return err
	}
	v = goutil.DereferenceValue(v)
	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			return info.typeError
		}
		v.Set(reflect.ValueOf(bodyBytes))
		return nil
	case reflect.String:
		v.Set(reflect.ValueOf(goutil.BytesToString(bodyBytes)))
		return nil
	default:
		return info.typeError
	}
}

func (p *paramInfo) bindPath(info *tagInfo, expr *tagexpr.TagExpr, pathParams PathParams) (bool, error) {
	if pathParams == nil {
		return false, nil
	}
	r, found := pathParams.Get(info.paramName)
	if !found {
		if info.required {
			return false, info.requiredError
		}
		return false, nil
	}
	return true, p.bindStringSlice(info, expr, []string{r})
}

func (p *paramInfo) bindQuery(info *tagInfo, expr *tagexpr.TagExpr, queryValues url.Values) (bool, error) {
	return p.bindMapStrings(info, expr, queryValues)
}

func (p *paramInfo) bindHeader(info *tagInfo, expr *tagexpr.TagExpr, header http.Header) (bool, error) {
	return p.bindMapStrings(info, expr, header)
}

func (p *paramInfo) bindCookie(info *tagInfo, expr *tagexpr.TagExpr, cookies []*http.Cookie) error {
	var r []string
	for _, c := range cookies {
		if c.Name == info.paramName {
			r = append(r, c.Value)
		}
	}
	if len(r) == 0 {
		if info.required {
			return info.requiredError
		}
		return nil
	}
	return p.bindStringSlice(info, expr, r)
}

func (p *paramInfo) bindOrRequireBody(info *tagInfo, expr *tagexpr.TagExpr, bodyCodec codec, bodyString string, postForm map[string][]string) (bool, error) {
	switch bodyCodec {
	case bodyForm:
		return p.bindMapStrings(info, expr, postForm)
	case bodyJSON:
		return p.checkRequireJSON(info, expr, bodyString, false)
	case bodyProtobuf:
		err := p.checkRequireProtobuf(info, expr, false)
		return err == nil, err
	default:
		return false, info.contentTypeError
	}
}

func (p *paramInfo) checkRequireProtobuf(info *tagInfo, expr *tagexpr.TagExpr, checkOpt bool) error {
	if checkOpt && !info.required {
		v, err := p.getField(expr, false)
		if err != nil || !v.IsValid() {
			return info.requiredError
		}
	}
	return nil
}

func (p *paramInfo) checkRequireJSON(info *tagInfo, expr *tagexpr.TagExpr, bodyString string, checkOpt bool) (bool, error) {
	var requiredError error
	if checkOpt || info.required { // only return error if it's a required field
		requiredError = info.requiredError
	}

	if !gjson.Get(bodyString, info.namePath).Exists() {
		idx := strings.LastIndex(info.namePath, ".")
		// There should be a superior but it is empty, no error is reported
		if idx > 0 && !gjson.Get(bodyString, info.namePath[:idx]).Exists() {
			return true, nil
		}
		return false, requiredError
	}
	v, err := p.getField(expr, false)
	if err != nil || !v.IsValid() {
		return false, requiredError
	}
	return true, nil
}

func (p *paramInfo) bindMapStrings(info *tagInfo, expr *tagexpr.TagExpr, values map[string][]string) (bool, error) {
	r, ok := values[info.paramName]
	if !ok || len(r) == 0 {
		if info.required {
			return false, info.requiredError
		}
		return false, nil
	}
	return true, p.bindStringSlice(info, expr, r)
}

// NOTE: len(a)>0
func (p *paramInfo) bindStringSlice(info *tagInfo, expr *tagexpr.TagExpr, a []string) error {
	v, err := p.getField(expr, true)
	if err != nil || !v.IsValid() {
		return err
	}

	v = goutil.DereferenceValue(v)
	switch v.Kind() {
	case reflect.String:
		v.SetString(a[0])
		return nil

	case reflect.Bool:
		var bol bool
		bol, err = strconv.ParseBool(a[0])
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetBool(bol)
			return nil
		}
	case reflect.Float32:
		var f float64
		f, err = strconv.ParseFloat(a[0], 32)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetFloat(f)
			return nil
		}
	case reflect.Float64:
		var f float64
		f, err = strconv.ParseFloat(a[0], 64)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetFloat(f)
			return nil
		}
	case reflect.Int64, reflect.Int:
		var i int64
		i, err = strconv.ParseInt(a[0], 10, 64)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetInt(i)
			return nil
		}
	case reflect.Int32:
		var i int64
		i, err = strconv.ParseInt(a[0], 10, 32)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetInt(i)
			return nil
		}
	case reflect.Int16:
		var i int64
		i, err = strconv.ParseInt(a[0], 10, 16)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetInt(i)
			return nil
		}
	case reflect.Int8:
		var i int64
		i, err = strconv.ParseInt(a[0], 10, 8)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetInt(i)
			return nil
		}
	case reflect.Uint64, reflect.Uint:
		var u uint64
		u, err = strconv.ParseUint(a[0], 10, 64)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetUint(u)
			return nil
		}
	case reflect.Uint32:
		var u uint64
		u, err = strconv.ParseUint(a[0], 10, 32)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetUint(u)
			return nil
		}
	case reflect.Uint16:
		var u uint64
		u, err = strconv.ParseUint(a[0], 10, 16)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetUint(u)
			return nil
		}
	case reflect.Uint8:
		var u uint64
		u, err = strconv.ParseUint(a[0], 10, 8)
		if err == nil || (a[0] == "" && p.looseZeroMode) {
			v.SetUint(u)
			return nil
		}
	case reflect.Slice:
		vv, err := stringsToValue(v.Type().Elem(), a, p.looseZeroMode)
		if err == nil {
			v.Set(vv)
			return nil
		}
		fallthrough
	default:
		fn := typeUnmarshalFuncs[v.Type()]
		if fn != nil {
			vv, err := fn(a[0], p.looseZeroMode)
			if err == nil {
				v.Set(vv)
				return nil
			}
		}
	}
	return info.typeError
}

func (p *paramInfo) bindDefaultVal(expr *tagexpr.TagExpr, defaultValue []byte) (bool, error) {
	if defaultValue == nil {
		return false, nil
	}
	v, err := p.getField(expr, true)
	if err != nil || !v.IsValid() {
		return false, err
	}
	return true, jsonpkg.Unmarshal(defaultValue, v.Addr().Interface())
}

// setDefaultVal preprocess the default tags and store the parsed value
func (p *paramInfo) setDefaultVal() error {
	for _, info := range p.tagInfos {
		if info.paramIn != default_val {
			continue
		}

		defaultVal := info.paramName
		st := ameda.DereferenceType(p.structField.Type)
		switch st.Kind() {
		case reflect.String:
			p.defaultVal, _ = jsonpkg.Marshal(defaultVal)
			continue
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
			// escape single quote and double quote, replace single quote with double quote
			defaultVal = strings.Replace(defaultVal, `"`, `\"`, -1)
			defaultVal = strings.Replace(defaultVal, `\'`, specialChar, -1)
			defaultVal = strings.Replace(defaultVal, `'`, `"`, -1)
			defaultVal = strings.Replace(defaultVal, specialChar, `'`, -1)
		}
		p.defaultVal = ameda.UnsafeStringToBytes(defaultVal)
	}
	return nil
}

var errMismatch = errors.New("type mismatch")

func stringsToValue(t reflect.Type, a []string, emptyAsZero bool) (reflect.Value, error) {
	var i interface{}
	var err error
	var ptrDepth int
	elemKind := t.Kind()
	for elemKind == reflect.Ptr {
		t = t.Elem()
		elemKind = t.Kind()
		ptrDepth++
	}
	switch elemKind {
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
