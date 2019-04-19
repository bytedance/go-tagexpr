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

	"github.com/henrylee2cn/goutil/tpack"
)

// VM struct tag expression interpreter
type VM struct {
	tagName   string
	structJar map[int32]*structVM
	rw        sync.RWMutex
}

// structVM tag expression set of struct
type structVM struct {
	vm                    *VM
	name                  string
	fields                map[string]*fieldVM
	fieldsWithSubStructVM []*fieldVM
	exprs                 map[string]*Expr
	selectorList          []string
	ifaceTagExprGetters   []func(ptr uintptr) (*TagExpr, bool)
}

// fieldVM tag expression set of struct field
type fieldVM struct {
	structField            reflect.StructField
	offset                 uintptr
	ptrDeep                int
	elemType               reflect.Type
	elemKind               reflect.Kind
	zeroValue              interface{}
	valueGetter            func(uintptr) interface{}
	reflectValueGetter     func(uintptr) reflect.Value
	origin                 *structVM
	mapKeyStructVM         *structVM
	mapOrSliceElemStructVM *structVM
}

// New creates a tag expression interpreter that uses @tagName as the tag name.
func New(tagName string) *VM {
	return &VM{
		tagName:   tagName,
		structJar: make(map[int32]*structVM, 256),
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

// MustWarmUp is similar to WarmUp, but panic when error.
func (vm *VM) MustWarmUp(structOrStructPtr ...interface{}) {
	err := vm.WarmUp(structOrStructPtr...)
	if err != nil {
		panic(err)
	}
}

// Run returns the tag expression handler of the @structPtr.
// NOTE:
//  If the structure type has not been warmed up,
//  it will be slower when it is first called.
func (vm *VM) Run(structOrStructPtr interface{}) (*TagExpr, error) {
	u := tpack.Unpack(structOrStructPtr)
	if u.IsNil() {
		return nil, errors.New("cannot run nil data")
	}
	u = u.UnderlyingElem()
	tid := u.RuntimeTypeID()
	var err error
	vm.rw.RLock()
	s, ok := vm.structJar[tid]
	vm.rw.RUnlock()
	if !ok {
		vm.rw.Lock()
		s, ok = vm.structJar[tid]
		if !ok {
			s, err = vm.registerStructLocked(reflect.TypeOf(structOrStructPtr))
			if err != nil {
				vm.rw.Unlock()
				return nil, err
			}
		}
		vm.rw.Unlock()
	}
	return s.newTagExpr(u.Pointer()), nil
}

// MustRun is similar to Run, but panic when error.
func (vm *VM) MustRun(structPtr interface{}) *TagExpr {
	te, err := vm.Run(structPtr)
	if err != nil {
		panic(err)
	}
	return te
}

func (vm *VM) registerStructLocked(structType reflect.Type) (*structVM, error) {
	structType, err := vm.getStructType(structType)
	if err != nil {
		return nil, err
	}
	tid := tpack.RuntimeTypeID(structType)
	s, had := vm.structJar[tid]
	if had {
		return s, nil
	}
	s = vm.newStructVM()
	vm.structJar[tid] = s
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
			switch field.elemKind {
			case reflect.Struct:
				sub, err = vm.registerStructLocked(field.structField.Type)
				if err != nil {
					return nil, err
				}
				field.origin = sub
				s.copySubFields(field, sub)
			case reflect.Interface:
				s.setIfaceTagExprGetter(field)
			}
		case reflect.Float32, reflect.Float64,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			field.setFloatGetter()
		case reflect.String:
			field.setStringGetter()
		case reflect.Bool:
			field.setBoolGetter()
		case reflect.Array, reflect.Slice, reflect.Map:
			err = vm.registerSubStructLocked(field)
			if err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

func (vm *VM) registerSubStructLocked(field *fieldVM) error {
	a := make([]reflect.Type, 1, 2)
	a[0] = derefType(field.elemType.Elem())
	if field.elemKind == reflect.Map {
		a = append(a, derefType(field.elemType.Key()))
	}
	for i, t := range a {
		if t.Kind() != reflect.Struct {
			continue
		}
		s, err := vm.registerStructLocked(t)
		if err != nil {
			return err
		}
		if len(s.selectorList) > 0 || len(s.ifaceTagExprGetters) > 0 {
			if i == 0 {
				field.mapOrSliceElemStructVM = s
			} else {
				field.mapKeyStructVM = s
			}
			field.origin.fieldsWithSubStructVM = append(field.origin.fieldsWithSubStructVM, field)
		}
	}
	field.setLengthGetter()
	return nil
}

func (vm *VM) newStructVM() *structVM {
	return &structVM{
		vm:                    vm,
		fields:                make(map[string]*fieldVM, 16),
		fieldsWithSubStructVM: make([]*fieldVM, 0, 4),
		exprs:                 make(map[string]*Expr, 64),
		selectorList:          make([]string, 0, 64),
	}
}

func (s *structVM) newFieldVM(structField reflect.StructField) (*fieldVM, error) {
	f := &fieldVM{
		structField: structField,
		offset:      structField.Offset,
		origin:      s,
	}
	err := f.parseExprs(structField.Tag.Get(s.vm.tagName))
	if err != nil {
		return nil, err
	}
	s.fields[f.structField.Name] = f
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
	f.reflectValueGetter = f.packRawFrom
	return f, nil
}

func (s *structVM) copySubFields(field *fieldVM, sub *structVM) {
	nameSpace := field.structField.Name
	offset := field.offset
	ptrDeep := field.ptrDeep
	for k, v := range sub.fields {
		valueGetter := v.valueGetter
		reflectValueGetter := v.reflectValueGetter
		f := &fieldVM{
			structField: v.structField,
			offset:      offset + v.offset,
			origin:      v.origin,
		}
		if valueGetter != nil {
			if ptrDeep == 0 {
				f.valueGetter = func(ptr uintptr) interface{} {
					return valueGetter(ptr + field.offset)
				}
				f.reflectValueGetter = func(ptr uintptr) reflect.Value {
					return reflectValueGetter(ptr + field.offset)
				}
			} else {
				f.valueGetter = func(ptr uintptr) interface{} {
					newFieldVM := reflect.NewAt(field.structField.Type, unsafe.Pointer(ptr+field.offset))
					for i := 0; i < ptrDeep; i++ {
						newFieldVM = newFieldVM.Elem()
					}
					if newFieldVM.IsNil() {
						return nil
					}
					return valueGetter(uintptr(newFieldVM.Pointer()))
				}
				f.reflectValueGetter = func(ptr uintptr) reflect.Value {
					newFieldVM := reflect.NewAt(field.structField.Type, unsafe.Pointer(ptr+field.offset))
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

func (f *fieldVM) elemPtr(ptr uintptr) uintptr {
	ptr = ptr + f.offset
	for i := f.ptrDeep; i > 0; i-- {
		ptr = uintptrElem(ptr)
	}
	return ptr
}

func (f *fieldVM) packRawFrom(ptr uintptr) reflect.Value {
	return reflect.NewAt(f.structField.Type, unsafe.Pointer(ptr+f.offset)).Elem()
}

func (f *fieldVM) packElemFrom(ptr uintptr) reflect.Value {
	return reflect.NewAt(f.elemType, unsafe.Pointer(f.elemPtr(ptr))).Elem()
}

func (s *structVM) setIfaceTagExprGetter(f *fieldVM) {
	s.ifaceTagExprGetters = append(s.ifaceTagExprGetters, func(ptr uintptr) (*TagExpr, bool) {
		v := f.packElemFrom(ptr)
		if !v.IsValid() || v.IsNil() {
			return nil, false
		}
		te, ok := s.vm.runFromValue(v)
		if !ok {
			return nil, false
		}
		return te, true
	})
}

func (vm *VM) runFromValue(v reflect.Value) (*TagExpr, bool) {
	u := tpack.From(v).UnderlyingElem()
	if u.Kind() != reflect.Struct {
		return nil, false
	}
	tid := u.RuntimeTypeID()
	var err error
	vm.rw.RLock()
	s, ok := vm.structJar[tid]
	vm.rw.RUnlock()
	if !ok {
		vm.rw.Lock()
		s, ok = vm.structJar[tid]
		if !ok {
			s, err = vm.registerStructLocked(v.Elem().Type())
			if err != nil {
				vm.rw.Unlock()
				return nil, false
			}
		}
		vm.rw.Unlock()
	}
	return s.newTagExpr(u.Pointer()), true
}

func (f *fieldVM) setFloatGetter() {
	if f.ptrDeep == 0 {
		f.valueGetter = func(ptr uintptr) interface{} {
			return getFloat64(f.elemKind, ptr+f.offset)
		}
	} else {
		f.valueGetter = func(ptr uintptr) interface{} {
			v := f.packElemFrom(ptr)
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
			return *(*bool)(unsafe.Pointer(ptr + f.offset))
		}
	} else {
		f.valueGetter = func(ptr uintptr) interface{} {
			v := f.packElemFrom(ptr)
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
			return *(*string)(unsafe.Pointer(ptr + f.offset))
		}
	} else {
		f.valueGetter = func(ptr uintptr) interface{} {
			v := f.packElemFrom(ptr)
			if v.IsValid() {
				return v.String()
			}
			return nil
		}
	}
}

func (f *fieldVM) setLengthGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		v := f.packElemFrom(ptr)
		if v.IsValid() {
			return v.Interface()
		}
		return nil
	}
}

func (f *fieldVM) setUnsupportGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		raw := f.packRawFrom(ptr)
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
		selector := f.structField.Name
		f.origin.exprs[selector] = expr
		f.origin.selectorList = append(f.origin.selectorList, selector)
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
					selector = f.structField.Name
				default:
					selector = f.structField.Name + "@" + selector
				}
				if _, had := f.origin.exprs[selector]; had {
					return fmt.Errorf("duplicate expression name: %s", selector)
				}
				exprStr = strings.TrimSpace((*subtag)[idx+1:])
				if exprStr != "" {
					if expr, err := parseExpr(exprStr); err == nil {
						f.origin.exprs[selector] = expr
						f.origin.selectorList = append(f.origin.selectorList, selector)
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
		sub: make(map[string]*TagExpr, 8),
	}
	return te
}

// TagExpr struct tag expression evaluator
type TagExpr struct {
	s   *structVM
	ptr uintptr
	sub map[string]*TagExpr
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
//  If the expression value is not 0, '' or nil, return true.
func (t *TagExpr) EvalBool(exprSelector string) bool {
	return FakeBool(t.Eval(exprSelector))
}

// FakeBool fakes any type as a boolean.
func FakeBool(v interface{}) bool {
	switch r := v.(type) {
	case float64:
		return r != 0
	case string:
		return r != ""
	case bool:
		return r
	case nil:
		return false
	default:
		return true
	}
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
	dir, base := splitFieldSelector(exprSelector)
	targetTagExpr, err := t.checkout(dir)
	if err != nil {
		return nil
	}
	return expr.run(base, targetTagExpr)
}

// Range loop through each tag expression.
// When fn returns false, interrupt traversal and return false.
// NOTE:
//  eval result types: float64, string, bool, nil
func (t *TagExpr) Range(fn func(exprSelector string, eval func() interface{}) bool) bool {
	if list := t.s.selectorList; len(list) > 0 {
		exprs := t.s.exprs
		for _, exprSelector := range list {
			if !fn(exprSelector, func() interface{} {
				dir, base := splitFieldSelector(exprSelector)
				targetTagExpr, err := t.checkout(dir)
				if err != nil {
					return nil
				}
				return exprs[exprSelector].run(base, targetTagExpr)
			}) {
				return false
			}
		}
	}

	if list := t.s.ifaceTagExprGetters; len(list) > 0 {
		var te *TagExpr
		var ok bool
		ptr := t.ptr
		for _, getter := range list {
			if te, ok = getter(ptr); ok {
				if !te.Range(fn) {
					return false
				}
			}
		}
	}

	if list := t.s.fieldsWithSubStructVM; len(list) > 0 {
		for _, f := range list {
			v := f.packElemFrom(t.ptr)

			if f.elemKind == reflect.Map {
				iter := v.MapRange()
				for iter.Next() {
					if f.mapKeyStructVM != nil {
						ptr := tpack.From(derefValue(iter.Key())).Pointer()
						if ptr == 0 {
							continue
						}
						if !f.mapKeyStructVM.newTagExpr(ptr).Range(fn) {
							return false
						}
					}
					if f.mapOrSliceElemStructVM != nil {
						ptr := tpack.From(derefValue(iter.Value())).Pointer()
						if ptr == 0 {
							continue
						}
						if !f.mapOrSliceElemStructVM.newTagExpr(ptr).Range(fn) {
							return false
						}
					}
				}

			} else {

				// slice or array
				for i := v.Len() - 1; i >= 0; i-- {
					ptr := tpack.From(derefValue(v.Index(i))).Pointer()
					if ptr == 0 {
						continue
					}
					if !f.mapOrSliceElemStructVM.newTagExpr(ptr).Range(fn) {
						return false
					}
				}
			}
		}
	}

	return true
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

var errFieldSelector = errors.New("field selector does not exist")

func (t *TagExpr) checkout(fs string) (*TagExpr, error) {
	if fs == "" {
		return t, nil
	}
	subTagExpr, ok := t.sub[fs]
	if ok {
		return subTagExpr, nil
	}
	f, ok := t.s.fields[fs]
	if !ok {
		return nil, errFieldSelector
	}
	subTagExpr = f.origin.newTagExpr(f.elemPtr(t.ptr))
	t.sub[fs] = subTagExpr
	return subTagExpr, nil
}

func (t *TagExpr) getValue(fieldSelector string, subFields []interface{}) (v interface{}) {
	f := t.s.fields[fieldSelector]
	if f == nil {
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

func splitFieldSelector(selector string) (dir, base string) {
	idx := strings.LastIndex(selector, "@")
	if idx != -1 {
		selector = selector[:idx]
	}
	idx = strings.LastIndex(selector, ".")
	if idx != -1 {
		return selector[:idx], selector[idx+1:]
	}
	return "", selector
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
	if !elem.IsValid() || !raw.IsValid() {
		return nil
	}
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
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func uintptrElem(ptr uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(ptr))
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}
