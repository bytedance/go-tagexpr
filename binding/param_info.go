package binding

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/bytedance/go-tagexpr"
	"github.com/henrylee2cn/goutil"
)

type paramInfo struct {
	fieldSelector string
	in            uint8
	name          string
	required      bool

	requiredError, typeError, cannotError error
}

func (p *paramInfo) getField(expr *tagexpr.TagExpr) (reflect.Value, error) {
	v := expr.Field(p.fieldSelector, true)
	if v.IsValid() {
		return v, nil
	}
	if p.required {
		return reflect.Value{}, p.cannotError
	}
	return reflect.Value{}, nil
}

func (p *paramInfo) bindRawBody(expr *tagexpr.TagExpr, bodyBytes []byte) error {
	if len(bodyBytes) == 0 {
		if p.required {
			return p.requiredError
		}
		return nil
	}
	v, err := p.getField(expr)
	if err != nil || !v.IsValid() {
		return err
	}
	v = derefValue(v)
	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			return p.typeError
		}
		v.Set(reflect.ValueOf(bodyBytes))
		return nil
	case reflect.String:
		v.Set(reflect.ValueOf(goutil.BytesToString(bodyBytes)))
		return nil
	default:
		return p.typeError
	}
}

func (p *paramInfo) bindQuery(expr *tagexpr.TagExpr, queryValues url.Values) error {
	r, ok := queryValues[p.name]
	if !ok || len(r) == 0 {
		if p.required {
			return p.requiredError
		}
		return nil
	}
	return p.bindStringSlice(expr, r)
}

func (p *paramInfo) bindHeader(expr *tagexpr.TagExpr, header http.Header) error {
	r, ok := header[p.name]
	if !ok || len(r) == 0 {
		if p.required {
			return p.requiredError
		}
		return nil
	}
	return p.bindStringSlice(expr, r)
}

func (p *paramInfo) bindCookie(expr *tagexpr.TagExpr, cookies []*http.Cookie) error {
	var r []string
	for _, c := range cookies {
		if c.Name == p.name {
			r = append(r, c.Value)
		}
	}
	if len(r) == 0 {
		if p.required {
			return p.requiredError
		}
		return nil
	}
	return p.bindStringSlice(expr, r)
}

func (p *paramInfo) bindStringSlice(expr *tagexpr.TagExpr, a []string) error {
	v, err := p.getField(expr)
	if err != nil || !v.IsValid() {
		return err
	}
	return p.setStringSlice(v, a)
}

func (p *paramInfo) setStringSlice(v reflect.Value, a []string) error {
	v = derefValue(v)
	switch v.Kind() {
	case reflect.String:
		v.Set(reflect.ValueOf(a[0]))
		return nil

	case reflect.Int64, reflect.Int:
		i, err := strconv.ParseInt(a[0], 10, 64)
		if err == nil {
			v.SetInt(i)
			return nil
		}
	case reflect.Int32:
		i, err := strconv.ParseInt(a[0], 10, 32)
		if err == nil {
			v.SetInt(i)
			return nil
		}
	case reflect.Int16:
		i, err := strconv.ParseInt(a[0], 10, 16)
		if err == nil {
			v.SetInt(i)
			return nil
		}
	case reflect.Int8:
		i, err := strconv.ParseInt(a[0], 10, 8)
		if err == nil {
			v.SetInt(i)
			return nil
		}

	case reflect.Uint64, reflect.Uint:
		i, err := strconv.ParseUint(a[0], 10, 64)
		if err == nil {
			v.SetUint(i)
			return nil
		}
	case reflect.Uint32:
		i, err := strconv.ParseUint(a[0], 10, 32)
		if err == nil {
			v.SetUint(i)
			return nil
		}
	case reflect.Uint16:
		i, err := strconv.ParseUint(a[0], 10, 16)
		if err == nil {
			v.SetUint(i)
			return nil
		}
	case reflect.Uint8:
		i, err := strconv.ParseUint(a[0], 10, 8)
		if err == nil {
			v.SetUint(i)
			return nil
		}

	case reflect.Slice:
		tt := v.Type().Elem()
		vv, err := stringsToValue(tt.Kind(), a)
		if err == nil {
			v.Set(vv)
			return nil
		}
	}

	return p.typeError
}

// func (p *paramInfo) newError(errStr string) error {
// 	return errors.New("field type does not match binding data: " + p.fieldSelector)
// }
