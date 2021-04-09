// The MIT License (MIT)

// Copyright (c) 2016 Josh Baker

// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package gjson

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/henrylee2cn/ameda"
	"github.com/tidwall/gjson"

	"github.com/bytedance/go-tagexpr/v2/binding"
)

var fieldsmu sync.RWMutex
var fields = make(map[uintptr]map[string]int)

func init() {
	gjson.DisableModifiers = true
}

// UseJSONUnmarshaler reset the JSON Unmarshaler of binding.
func UseJSONUnmarshaler() {
	binding.ResetJSONUnmarshaler(unmarshal)
}

// unmarshal unmarshal JSON, old version compatible.
func unmarshal(data []byte, v interface{}) error {
	val, ok := v.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(v)
	}
	assign(gjson.Parse(ameda.UnsafeBytesToString(data)), val)
	return nil
}

// assign unmarshal
func assign(jsval gjson.Result, goval reflect.Value) {
	if jsval.Type == gjson.Null {
		return
	}
	t := goval.Type()
	switch goval.Kind() {
	default:
	case reflect.Ptr:
		if !goval.IsNil() {
			newval := reflect.New(goval.Elem().Type())
			assign(jsval, newval.Elem())
			goval.Elem().Set(newval.Elem())
		} else {
			newval := reflect.New(t.Elem())
			assign(jsval, newval.Elem())
			goval.Set(newval)
		}
	case reflect.Struct:
		runtimeTypeID := ameda.ValueFrom(goval).RuntimeTypeID()
		fieldsmu.RLock()
		sf := fields[runtimeTypeID]
		fieldsmu.RUnlock()
		if sf == nil {
			fieldsmu.Lock()
			sf = make(map[string]int)
			numField := t.NumField()
			for i := 0; i < numField; i++ {
				f := t.Field(i)
				tag := strings.Split(f.Tag.Get("json"), ",")[0]
				if tag != "-" {
					if tag != "" {
						sf[tag] = i
						sf[f.Name] = i
					} else {
						sf[f.Name] = i
					}
				}
			}
			fields[runtimeTypeID] = sf
			fieldsmu.Unlock()
		}
		jsval.ForEach(func(key, value gjson.Result) bool {
			if idx, ok := sf[key.Str]; ok {
				f := goval.Field(idx)
				if f.CanSet() {
					assign(value, f)
				}
			}
			return true
		})
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 && jsval.Type == gjson.String {
			data, _ := base64.StdEncoding.DecodeString(jsval.String())
			goval.Set(reflect.ValueOf(data))
		} else {
			jsvals := jsval.Array()
			slice := reflect.MakeSlice(t, len(jsvals), len(jsvals))
			for i := 0; i < len(jsvals); i++ {
				assign(jsvals[i], slice.Index(i))
			}
			goval.Set(slice)
		}
	case reflect.Array:
		i, n := 0, goval.Len()
		jsval.ForEach(func(_, value gjson.Result) bool {
			if i == n {
				return false
			}
			assign(value, goval.Index(i))
			i++
			return true
		})
	case reflect.Map:
		if jsval.Type == gjson.JSON && t.Key().Kind() == reflect.String {
			if t.Elem().Kind() == reflect.Interface {
				goval.Set(reflect.ValueOf(jsval.Value()))
			} else {
				if goval.IsNil() {
					goval.Set(reflect.MakeMap(t))
				}
				valType := t.Elem()
				jsval.ForEach(func(key, value gjson.Result) bool {
					val := reflect.New(valType)
					assign(value, val)
					goval.SetMapIndex(reflect.ValueOf(key.String()), val.Elem())
					return true
				})
			}
		}
	case reflect.Interface:
		goval.Set(reflect.ValueOf(jsval.Value()))
	case reflect.Bool:
		goval.SetBool(jsval.Bool())
	case reflect.Float32, reflect.Float64:
		goval.SetFloat(jsval.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		goval.SetInt(jsval.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		goval.SetUint(jsval.Uint())
	case reflect.String:
		goval.SetString(jsval.String())
	}
	if len(t.PkgPath()) > 0 {
		v := goval.Addr()
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(json.Unmarshaler); ok {
				u.UnmarshalJSON([]byte(jsval.Raw))
			}
		}
	}
}
