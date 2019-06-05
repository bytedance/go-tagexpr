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

package jsonparam

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

var fieldsmu sync.RWMutex
var fields = make(map[string]map[string]int)

// Assign unmarshal
func Assign(jsval gjson.Result, goval reflect.Value) {
	if jsval.Type == gjson.Null {
		return
	}
	switch goval.Kind() {
	default:
	case reflect.Ptr:
		if !goval.IsNil() {
			newval := reflect.New(goval.Elem().Type())
			Assign(jsval, newval.Elem())
			goval.Elem().Set(newval.Elem())
		} else {
			newval := reflect.New(goval.Type().Elem())
			Assign(jsval, newval.Elem())
			goval.Set(newval)
		}
	case reflect.Struct:
		fieldsmu.RLock()
		sf := fields[goval.Type().String()]
		fieldsmu.RUnlock()
		if sf == nil {
			fieldsmu.Lock()
			sf = make(map[string]int)
			for i := 0; i < goval.Type().NumField(); i++ {
				f := goval.Type().Field(i)
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
			fields[goval.Type().String()] = sf
			fieldsmu.Unlock()
		}
		jsval.ForEach(func(key, value gjson.Result) bool {
			if idx, ok := sf[key.Str]; ok {
				f := goval.Field(idx)
				if f.CanSet() {
					Assign(value, f)
				}
			}
			return true
		})
	case reflect.Slice:
		if goval.Type().Elem().Kind() == reflect.Uint8 && jsval.Type == gjson.String {
			data, _ := base64.StdEncoding.DecodeString(jsval.String())
			goval.Set(reflect.ValueOf(data))
		} else {
			jsvals := jsval.Array()
			slice := reflect.MakeSlice(goval.Type(), len(jsvals), len(jsvals))
			for i := 0; i < len(jsvals); i++ {
				Assign(jsvals[i], slice.Index(i))
			}
			goval.Set(slice)
		}
	case reflect.Array:
		i, n := 0, goval.Len()
		jsval.ForEach(func(_, value gjson.Result) bool {
			if i == n {
				return false
			}
			Assign(value, goval.Index(i))
			i++
			return true
		})
	case reflect.Map:
		if goval.Type().Key().Kind() == reflect.String && goval.Type().Elem().Kind() == reflect.Interface {
			goval.Set(reflect.ValueOf(jsval.Value()))
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
	if len(goval.Type().PkgPath()) > 0 {
		v := goval.Addr()
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(json.Unmarshaler); ok {
				u.UnmarshalJSON([]byte(jsval.Raw))
			}
		}
	}
}
