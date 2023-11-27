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
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/andeya/ameda"
	"github.com/andeya/goutil"
	"github.com/bytedance/go-tagexpr/v2/binding"
	"github.com/bytedance/go-tagexpr/v2/binding/gjson/internal/caching"
	"github.com/bytedance/go-tagexpr/v2/binding/gjson/internal/rt"
	gjson "github.com/bytedance/go-tagexpr/v2/binding/tidwall_gjson"
)

var (
	programCache         = caching.CreateProgramCache()
	unmarshalerInterface = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
)

func init() {
	gjson.DisableModifiers = true
}

// UseJSONUnmarshaler reset the JSON Unmarshaler of binding.
func UseJSONUnmarshaler() {
	binding.Default().ResetJSONUnmarshaler(Unmarshal)
}

// Unmarshal Unmarshal JSON, old version compatible.
func Unmarshal(data []byte, v interface{}) error {
	val, ok := v.(reflect.Value)
	if !ok {
		val = reflect.ValueOf(v)
	}
	return assign(gjson.ParseBytes(data), val)
}

// assign Unmarshal
func assign(jsval gjson.Result, goval reflect.Value) (err error) {
	if jsval.Type == gjson.Null {
		return nil
	}
	t := goval.Type()
	switch goval.Kind() {
	default:
	case reflect.Ptr:
		if !ameda.InitPointer(goval) {
			return errors.New("v cannot be set")
		}
		newval := ameda.DereferencePtrValue(goval)
		if err = assign(jsval, newval); err != nil {
			return err
		}
	case reflect.Struct:
		sf := getFiledInfo(t)
		jsval.ForEach(func(key, value gjson.Result) bool {
			if idx, ok := sf[key.Str]; ok {
				f := fieldByIndex(goval, idx)
				if f.CanSet() {
					if err = assign(value, f); err != nil {
						return false
					}
				}
			}
			return true
		})
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 && jsval.Type == gjson.String {
			var data []byte
			data, err = base64.StdEncoding.DecodeString(jsval.String())
			if err != nil {
				return err
			}
			goval.Set(reflect.ValueOf(data))
		} else {
			if !jsval.IsArray() {
				// canAddr: true, implement unmarshaler : true   -> continue
				// canAddr: true, implement unmarshaler : false  -> return
				// canAddr: false, implement unmarshaler : true  -> return
				// canAddr: false, implement unmarshaler : false -> return
				if !goval.CanAddr() || !goval.Addr().Type().Implements(unmarshalerInterface) {
					return nil
				}
			}
			jsvals := jsval.Array()
			slice := reflect.MakeSlice(t, len(jsvals), len(jsvals))
			for i := 0; i < len(jsvals); i++ {
				if err = assign(jsvals[i], slice.Index(i)); err != nil {
					return err
				}
			}
			goval.Set(slice)
		}
	case reflect.Array:
		i, n := 0, goval.Len()
		jsval.ForEach(func(_, value gjson.Result) bool {
			if i == n {
				return false
			}
			if err = assign(value, goval.Index(i)); err != nil {
				return false
			}
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
				keyType := goval.Type().Key()
				switch keyType.Kind() {
				case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
					reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				default:
					return fmt.Errorf("gjson: unsupported type: %s", keyType)
				}
				jsval.ForEach(func(key, value gjson.Result) bool {
					val := reflect.New(valType)
					if err = assign(value, val); err != nil {
						return false
					}
					goval.SetMapIndex(reflect.ValueOf(key.String()).Convert(keyType), val.Elem())
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
				if err = u.UnmarshalJSON([]byte(jsval.Raw)); err != nil {
					return err
				}
			}
		}
	}
	return err
}

func getFiledInfo(t reflect.Type) map[string][]int {
	vtr := rt.UnpackType(t)
	filedInfo := programCache.Get(vtr)
	if filedInfo == nil {
		pp, err := programCache.Compute(vtr, computeTypeInfo)
		if err == nil {
			return pp.(map[string][]int)
		}
		filedInfo, _ = computeTypeInfo(vtr)
	}
	return filedInfo.(map[string][]int)
}

func computeTypeInfo(vtr *rt.GoType) (interface{}, error) {
	t := vtr.Pack()
	sf := make(map[string][]int)
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		f := t.Field(i)
		if !f.Anonymous && !goutil.IsExportedName(f.Name) {
			continue
		}
		tag := getJsonTag(f.Tag)
		if tag == "-" {
			continue
		}
		if tag != "" {
			sf[tag] = []int{i}
		} else if f.Anonymous {
			if findAnonymous(ameda.DereferenceType(f.Type), []int{i}, sf, 20) {
				continue
			}
		}
		if tag != f.Name {
			sf[f.Name] = []int{i}
		}
	}
	return sf, nil
}

func getJsonTag(tag reflect.StructTag) string {
	return strings.Split(tag.Get("json"), ",")[0]
}

func findAnonymous(t reflect.Type, i []int, sf map[string][]int, depth int) bool {
	depth--
	if depth < 0 {
		return true
	}
	if t.Kind() == reflect.Struct {
		subNumField := t.NumField()
		for ii := 0; ii < subNumField; ii++ {
			ff := t.Field(ii)
			subTag := getJsonTag(ff.Tag)
			if subTag == "-" {
				continue
			}
			a := append(i, ii)
			if subTag != "" {
				sf[subTag] = a
			} else if ff.Anonymous {
				tt := ameda.DereferenceType(ff.Type)
				if tt.String() == t.String() {
					continue
				}
				if findAnonymous(tt, a, sf, depth) {
					continue
				}
			}
			if subTag != ff.Name {
				sf[ff.Name] = a
			}
		}
		return true
	}
	return false
}

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	if len(index) == 1 {
		return v.Field(index[0])
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	v = v.Field(index[0])
	for _, x := range index[1:] {
		for v.Kind() == reflect.Ptr && v.Type().Elem().Kind() == reflect.Struct {
			if v.IsNil() {
				if v.CanSet() {
					ptrDepth := 0
					t := v.Type()
					for t.Kind() == reflect.Ptr {
						t = t.Elem()
						ptrDepth++
					}
					v.Set(ameda.ReferenceValue(reflect.New(t), ptrDepth-1))
					v = ameda.DereferencePtrValue(v)
				} else {
					return reflect.Value{}
				}
			} else {
				v = ameda.DereferencePtrValue(v)
			}
		}
		v = v.Field(x)
	}
	return v
}
