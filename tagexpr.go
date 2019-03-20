// Package tagexpr is an interesting go struct tag expression syntax for field validation, etc.
//
// Copyright 2019 Bytedance Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tagexpr

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// VM struct tag expression interpreter
type VM struct {
	tagName   string
	structJar map[string]*structVM
	rw        sync.RWMutex
}

// structVM tag expression set of struct
type structVM struct {
	vm           *VM
	name         string
	fields       map[string]*fieldVM
	exprs        map[string]*Expr
	selectorList []string
}

// fieldVM tag expression set of struct field
type fieldVM struct {
	reflect.StructField
	ptrDeep            int
	elemType           reflect.Type
	elemKind           reflect.Kind
	zeroValue          interface{}
	host               *structVM
	valueGetter        func(uintptr) interface{}
	reflectValueGetter func(uintptr) reflect.Value
}

// New creates a tag expression interpreter that uses @tagName as the tag name.
func New(tagName string) *VM {
	return &VM{
		tagName:   tagName,
		structJar: make(map[string]*structVM, 256),
	}
}

// WarmUp preheating some interpreters of the struct type in batches,
// to improve the performance of the vm.Run.
func (vm *VM) WarmUp(structOrStructPtr ...interface{}) error {
	vm.rw.Lock()
	defer vm.rw.Unlock()
	for _, v := range structOrStructPtr {
		if v == nil {
			return errors.New("cannot warn up nil interface")
		}
		_, err := vm.registerStructLocked(reflect.TypeOf(v))
		if err != nil {
			return err
		}
	}
	return nil
}

// Run returns the tag expression handler of the @structPtr.
// NOTE:
//  If the structure type has not been warmed up,
//  it will be slower when it is first called.
func (vm *VM) Run(structPtr interface{}) (*TagExpr, error) {
	if structPtr == nil {
		return nil, errors.New("cannot run nil interface")
	}
	v := reflect.ValueOf(structPtr)
	if v.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("not structure pointer: %s", v.Type().String())
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return nil, fmt.Errorf("not structure pointer: %s", v.Type().String())
	}
	t := elem.Type()
	tname := t.String()
	var err error
	vm.rw.RLock()
	s, ok := vm.structJar[tname]
	vm.rw.RUnlock()
	if !ok {
		vm.rw.Lock()
		s, ok = vm.structJar[tname]
		if !ok {
			s, err = vm.registerStructLocked(t)
			if err != nil {
				vm.rw.Unlock()
				return nil, err
			}
		}
		vm.rw.Unlock()
	}
	return s.newTagExpr(v.Pointer()), nil
}

func (vm *VM) registerStructLocked(structType reflect.Type) (*structVM, error) {
	structType, err := vm.getStructType(structType)
	if err != nil {
		return nil, err
	}
	structTypeName := structType.String()
	s, had := vm.structJar[structTypeName]
	if had {
		return s, nil
	}
	s = vm.newStructVM()
	vm.structJar[structTypeName] = s
	var numField = structType.NumField()
	var structField reflect.StructField
	var sub *structVM
	for i := 0; i < numField; i++ {
		structField = structType.Field(i)
		field, err := s.newFieldVM(structField)
		if err != nil {
			return nil, err
		}
		switch field.elemKind {
		default:
			field.setUnsupportGetter()
			if field.elemKind == reflect.Struct {
				sub, err = vm.registerStructLocked(field.Type)
				if err != nil {
					return nil, err
				}
				s.copySubFields(field, sub)
			}
		case reflect.Float32, reflect.Float64,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			field.setFloatGetter()
		case reflect.String:
			field.setStringGetter()
		case reflect.Bool:
			field.setBoolGetter()
		case reflect.Map, reflect.Array, reflect.Slice:
			field.setLengthGetter()
		}
	}
	return s, nil
}

func (vm *VM) newStructVM() *structVM {
	return &structVM{
		vm:           vm,
		fields:       make(map[string]*fieldVM, 16),
		exprs:        make(map[string]*Expr, 64),
		selectorList: make([]string, 0, 64),
	}
}

func (s *structVM) newFieldVM(structField reflect.StructField) (*fieldVM, error) {
	f := &fieldVM{
		StructField: structField,
		host:        s,
	}
	err := f.parseExprs(structField.Tag.Get(s.vm.tagName))
	if err != nil {
		return nil, err
	}
	s.fields[f.Name] = f
	var t = structField.Type
	var ptrDeep int
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		ptrDeep++
	}
	f.ptrDeep = ptrDeep
	f.elemType = t
	f.elemKind = t.Kind()
	if f.ptrDeep == 0 {
		f.zeroValue = reflect.New(t).Elem().Interface()
	}
	f.reflectValueGetter = f.newRawFrom
	return f, nil
}

func (s *structVM) copySubFields(field *fieldVM, sub *structVM) {
	nameSpace := field.Name
	ptrDeep := field.ptrDeep
	for k, v := range sub.fields {
		valueGetter := v.valueGetter
		reflectValueGetter := v.reflectValueGetter
		f := &fieldVM{
			StructField: v.StructField,
			host:        v.host,
		}
		if valueGetter != nil {
			if ptrDeep == 0 {
				f.valueGetter = func(ptr uintptr) interface{} {
					return valueGetter(ptr + field.Offset)
				}
				f.reflectValueGetter = func(ptr uintptr) reflect.Value {
					return reflectValueGetter(ptr + field.Offset)
				}
			} else {
				f.valueGetter = func(ptr uintptr) interface{} {
					newFieldVM := reflect.NewAt(field.Type, unsafe.Pointer(ptr+field.Offset))
					for i := 0; i < ptrDeep; i++ {
						newFieldVM = newFieldVM.Elem()
					}
					if newFieldVM.IsNil() {
						return nil
					}
					return valueGetter(uintptr(newFieldVM.Pointer()))
				}
				f.reflectValueGetter = func(ptr uintptr) reflect.Value {
					newFieldVM := reflect.NewAt(field.Type, unsafe.Pointer(ptr+field.Offset))
					for i := 0; i < ptrDeep; i++ {
						newFieldVM = newFieldVM.Elem()
					}
					if newFieldVM.IsNil() {
						return reflect.Value{}
					}
					return reflectValueGetter(uintptr(newFieldVM.Pointer()))
				}
			}
		}
		s.fields[nameSpace+"."+k] = f
	}
	var selector string
	for k, v := range sub.exprs {
		selector = nameSpace + "." + k
		s.exprs[selector] = v
		s.selectorList = append(s.selectorList, selector)
	}
}

func (f *fieldVM) newRawFrom(ptr uintptr) reflect.Value {
	return reflect.NewAt(f.Type, unsafe.Pointer(ptr+f.Offset)).Elem()
}

func (f *fieldVM) newElemFrom(ptr uintptr) reflect.Value {
	v := f.newRawFrom(ptr)
	for i := 0; i < f.ptrDeep; i++ {
		v = v.Elem()
	}
	return v
}

func (f *fieldVM) setFloatGetter() {
	if f.ptrDeep == 0 {
		f.valueGetter = func(ptr uintptr) interface{} {
			return getFloat64(f.elemKind, ptr+f.Offset)
		}
	} else {
		f.valueGetter = func(ptr uintptr) interface{} {
			v := f.newElemFrom(ptr)
			if v.CanAddr() {
				return getFloat64(f.elemKind, v.UnsafeAddr())
			}
			return nil
		}
	}
}

func (f *fieldVM) setBoolGetter() {
	if f.ptrDeep == 0 {
		f.valueGetter = func(ptr uintptr) interface{} {
			return *(*bool)(unsafe.Pointer(ptr + f.Offset))
		}
	} else {
		f.valueGetter = func(ptr uintptr) interface{} {
			v := f.newElemFrom(ptr)
			if v.IsValid() {
				return v.Bool()
			}
			return nil
		}
	}
}

func (f *fieldVM) setStringGetter() {
	if f.ptrDeep == 0 {
		f.valueGetter = func(ptr uintptr) interface{} {
			return *(*string)(unsafe.Pointer(ptr + f.Offset))
		}
	} else {
		f.valueGetter = func(ptr uintptr) interface{} {
			v := f.newElemFrom(ptr)
			if v.IsValid() {
				return v.String()
			}
			return nil
		}
	}
}

func (f *fieldVM) setLengthGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		v := f.newElemFrom(ptr)
		if v.IsValid() {
			return v.Interface()
		}
		return nil
	}
}

func (f *fieldVM) setUnsupportGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		raw := f.newRawFrom(ptr)
		if safeIsNil(raw) {
			return nil
		}
		v := raw
		for i := 0; i < f.ptrDeep; i++ {
			v = v.Elem()
		}
		for v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		return anyValueGetter(raw, v)
	}
}

func (f *fieldVM) parseExprs(tag string) error {
	raw := tag
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	if tag[0] != '{' {
		expr, err := parseExpr(tag)
		if err != nil {
			return err
		}
		selector := f.Name
		f.host.exprs[selector] = expr
		f.host.selectorList = append(f.host.selectorList, selector)
		return nil
	}
	var subtag *string
	var idx int
	var selector, exprStr string
	for {
		subtag = readPairedSymbol(&tag, '{', '}')
		if subtag != nil {
			idx = strings.Index(*subtag, ":")
			if idx > 0 {
				selector = strings.TrimSpace((*subtag)[:idx])
				switch selector {
				case "":
					continue
				case "@":
					selector = f.Name
				default:
					selector = f.Name + "@" + selector
				}
				if _, had := f.host.exprs[selector]; had {
					return fmt.Errorf("duplicate expression name: %s", selector)
				}
				exprStr = strings.TrimSpace((*subtag)[idx+1:])
				if exprStr != "" {
					if expr, err := parseExpr(exprStr); err == nil {
						f.host.exprs[selector] = expr
						f.host.selectorList = append(f.host.selectorList, selector)
					} else {
						return err
					}
					trimLeftSpace(&tag)
					if tag == "" {
						return nil
					}
					continue
				}
			}
		}
		return fmt.Errorf("syntax incorrect: %q", raw)
	}
}

func (vm *VM) getStructType(t reflect.Type) (reflect.Type, error) {
	structType := t
	for structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("not structure pointer or structure: %s", t.String())
	}
	return structType, nil
}

func (s *structVM) newTagExpr(ptr uintptr) *TagExpr {
	te := &TagExpr{
		s:   s,
		ptr: ptr,
	}
	return te
}

// TagExpr struct tag expression evaluator
type TagExpr struct {
	s   *structVM
	ptr uintptr
}

// EvalFloat evaluate the value of the struct tag expression by the selector expression.
// NOTE:
//  If the expression value type is not float64, return 0.
func (t *TagExpr) EvalFloat(exprSelector string) float64 {
	r, _ := t.Eval(exprSelector).(float64)
	return r
}

// EvalString evaluate the value of the struct tag expression by the selector expression.
// NOTE:
//  If the expression value type is not string, return "".
func (t *TagExpr) EvalString(exprSelector string) string {
	r, _ := t.Eval(exprSelector).(string)
	return r
}

// EvalBool evaluate the value of the struct tag expression by the selector expression.
// NOTE:
//  If the expression value type is not bool, return false.
func (t *TagExpr) EvalBool(exprSelector string) bool {
	r, _ := t.Eval(exprSelector).(bool)
	return r
}

// Eval evaluate the value of the struct tag expression by the selector expression.
// NOTE:
//  format: fieldName, fieldName.exprName, fieldName1.fieldName2.exprName1
//  result types: float64, string, bool, nil
func (t *TagExpr) Eval(exprSelector string) interface{} {
	expr, ok := t.s.exprs[exprSelector]
	if !ok {
		// Compatible with single mode or the expression with the name @
		if strings.HasSuffix(exprSelector, "@") {
			exprSelector = exprSelector[:len(exprSelector)-1]
			if strings.HasSuffix(exprSelector, "@") {
				exprSelector = exprSelector[:len(exprSelector)-1]
			}
			expr, ok = t.s.exprs[exprSelector]
		}
		if !ok {
			return nil
		}
	}
	return expr.run(getFieldSelector(exprSelector), t)
}

// Range loop through each tag expression
// NOTE:
//  eval result types: float64, string, bool, nil
func (t *TagExpr) Range(fn func(exprSelector string, eval func() interface{}) bool) {
	exprs := t.s.exprs
	for _, exprSelector := range t.s.selectorList {
		if !fn(exprSelector, func() interface{} {
			return exprs[exprSelector].run(getFieldSelector(exprSelector), t)
		}) {
			return
		}
	}
}

// Field returns the field value specified by the selector.
// NOTE:
//  Return nil if the field is not exist
func (t *TagExpr) Field(fieldSelector string) interface{} {
	f, ok := t.s.fields[fieldSelector]
	if !ok {
		return nil
	}
	elem := f.reflectValueGetter(t.ptr)
	if !elem.IsValid() {
		return f.zeroValue
	}
	if elem.CanInterface() {
		return elem.Interface()
	}
	return nil
}

func (t *TagExpr) getValue(field string, subFields []interface{}) (v interface{}) {
	f, ok := t.s.fields[field]
	if !ok {
		return nil
	}
	if f.valueGetter == nil {
		return nil
	}
	v = f.valueGetter(t.ptr)
	if v == nil {
		return nil
	}
	if len(subFields) == 0 {
		return v
	}
	vv := reflect.ValueOf(v)
	var kind reflect.Kind
	for i, k := range subFields {
		kind = vv.Kind()
		for kind == reflect.Ptr || kind == reflect.Interface {
			vv = vv.Elem()
			kind = vv.Kind()
		}
		switch kind {
		case reflect.Slice, reflect.Array, reflect.String:
			if float, ok := k.(float64); ok {
				idx := int(float)
				if idx >= vv.Len() {
					return nil
				}
				vv = vv.Index(idx)
			} else {
				return nil
			}
		case reflect.Map:
			k := safeConvert(reflect.ValueOf(k), vv.Type().Key())
			if !k.IsValid() {
				return nil
			}
			vv = vv.MapIndex(k)
		case reflect.Struct:
			if float, ok := k.(float64); ok {
				idx := int(float)
				if idx < 0 || idx >= vv.NumField() {
					return nil
				}
				vv = vv.Field(idx)
			} else if str, ok := k.(string); ok {
				vv = vv.FieldByName(str)
			} else {
				return nil
			}
		default:
			if i < len(subFields)-1 {
				return nil
			}
		}
		if !vv.IsValid() {
			return nil
		}
	}
	raw := vv
	for vv.Kind() == reflect.Ptr || vv.Kind() == reflect.Interface {
		vv = vv.Elem()
	}
	return anyValueGetter(raw, vv)
}

func safeConvert(v reflect.Value, t reflect.Type) reflect.Value {
	defer func() { recover() }()
	return v.Convert(t)
}

var float64Type = reflect.TypeOf(float64(0))

func getFieldSelector(selector string) string {
	idx := strings.Index(selector, "@")
	if idx == -1 {
		return selector
	}
	return selector[:idx]
}

func getFloat64(kind reflect.Kind, ptr uintptr) interface{} {
	p := unsafe.Pointer(ptr)
	switch kind {
	case reflect.Float32:
		return float64(*(*float32)(p))
	case reflect.Float64:
		return *(*float64)(p)
	case reflect.Int:
		return float64(*(*int)(p))
	case reflect.Int8:
		return float64(*(*int8)(p))
	case reflect.Int16:
		return float64(*(*int16)(p))
	case reflect.Int32:
		return float64(*(*int32)(p))
	case reflect.Int64:
		return float64(*(*int64)(p))
	case reflect.Uint:
		return float64(*(*uint)(p))
	case reflect.Uint8:
		return float64(*(*uint8)(p))
	case reflect.Uint16:
		return float64(*(*uint16)(p))
	case reflect.Uint32:
		return float64(*(*uint32)(p))
	case reflect.Uint64:
		return float64(*(*uint64)(p))
	case reflect.Uintptr:
		return float64(*(*uintptr)(p))
	}
	return nil
}

func anyValueGetter(raw, elem reflect.Value) interface{} {
	// if !elem.IsValid() || !raw.IsValid() {
	// 	return nil
	// }
	kind := elem.Kind()
	switch kind {
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if elem.CanAddr() {
			return getFloat64(kind, elem.UnsafeAddr())
		}
		switch kind {
		case reflect.Float32, reflect.Float64:
			return elem.Float()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(elem.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return float64(elem.Uint())
		}
	case reflect.String:
		return elem.String()
	case reflect.Bool:
		return elem.Bool()
	}
	if raw.CanInterface() {
		return raw.Interface()
	}
	return nil
}

func safeIsNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}
