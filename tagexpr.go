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
	vm                         *VM
	name                       string
	fields                     map[string]*fieldVM
	fieldSelectorList          []string
	fieldsWithIndirectStructVM []*fieldVM
	exprs                      map[string]*Expr
	exprSelectorList           []string
	ifaceTagExprGetters        []func(ptr uintptr) (*TagExpr, bool)
}

// fieldVM tag expression set of struct field
type fieldVM struct {
	structField            reflect.StructField
	offset                 uintptr
	ptrDeep                int
	elemType               reflect.Type
	elemKind               reflect.Kind
	valueGetter            func(uintptr) interface{}
	reflectValueGetter     func(uintptr, bool) reflect.Value
	exprs                  map[string]*Expr
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
func (vm *VM) WarmUp(structOrStructPtrOrReflect ...interface{}) error {
	vm.rw.Lock()
	defer vm.rw.Unlock()
	var err error
	for _, v := range structOrStructPtrOrReflect {
		switch t := v.(type) {
		case nil:
			return errors.New("cannot warn up nil interface")
		case reflect.Type:
			_, err = vm.registerStructLocked(t)
		case reflect.Value:
			_, err = vm.registerStructLocked(t.Type())
		default:
			_, err = vm.registerStructLocked(reflect.TypeOf(v))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MustWarmUp is similar to WarmUp, but panic when error.
func (vm *VM) MustWarmUp(structOrStructPtrOrReflect ...interface{}) {
	err := vm.WarmUp(structOrStructPtrOrReflect...)
	if err != nil {
		panic(err)
	}
}

// Run returns the tag expression handler of the @structOrStructPtrOrReflectValue.
// NOTE:
//  If the structure type has not been warmed up,
//  it will be slower when it is first called.
func (vm *VM) Run(structOrStructPtrOrReflectValue interface{}) (*TagExpr, error) {
	var u tpack.U
	v, isReflectValue := structOrStructPtrOrReflectValue.(reflect.Value)
	if isReflectValue {
		u = tpack.From(v)
	} else {
		u = tpack.Unpack(structOrStructPtrOrReflectValue)
	}
	if u.IsNil() {
		return nil, errors.New("unsupport data: nil")
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
			if isReflectValue {
				s, err = vm.registerStructLocked(v.Type())
			} else {
				s, err = vm.registerStructLocked(reflect.TypeOf(structOrStructPtrOrReflectValue))
			}
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
func (vm *VM) MustRun(structOrStructPtrOrReflectValue interface{}) *TagExpr {
	te, err := vm.Run(structOrStructPtrOrReflectValue)
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
			err = vm.registerIndirectStructLocked(field)
			if err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

func (vm *VM) registerIndirectStructLocked(field *fieldVM) error {
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
		if len(s.exprSelectorList) > 0 || len(s.ifaceTagExprGetters) > 0 {
			if i == 0 {
				field.mapOrSliceElemStructVM = s
			} else {
				field.mapKeyStructVM = s
			}
			field.origin.fieldsWithIndirectStructVM = append(field.origin.fieldsWithIndirectStructVM, field)
		}
	}
	field.setLengthGetter()
	return nil
}

func (vm *VM) newStructVM() *structVM {
	return &structVM{
		vm:                         vm,
		fields:                     make(map[string]*fieldVM, 32),
		fieldSelectorList:          make([]string, 0, 32),
		fieldsWithIndirectStructVM: make([]*fieldVM, 0, 32),
		exprs:                      make(map[string]*Expr, 64),
		exprSelectorList:           make([]string, 0, 64),
	}
}

func (s *structVM) newFieldVM(structField reflect.StructField) (*fieldVM, error) {
	f := &fieldVM{
		structField: structField,
		offset:      structField.Offset,
		exprs:       make(map[string]*Expr, 8),
		origin:      s,
	}
	err := f.parseExprs(structField.Tag.Get(s.vm.tagName))
	if err != nil {
		return nil, err
	}
	fieldSelector := f.structField.Name
	s.fields[fieldSelector] = f
	s.fieldSelectorList = append(s.fieldSelectorList, fieldSelector)
	var t = structField.Type
	var ptrDeep int
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		ptrDeep++
	}
	f.ptrDeep = ptrDeep
	f.elemType = t
	f.elemKind = t.Kind()
	f.reflectValueGetter = func(ptr uintptr, initZero bool) reflect.Value {
		v := f.packRawFrom(ptr)
		if initZero {
			f.ensureInit(v)
		}
		return v
	}
	return f, nil
}

func (f *fieldVM) ensureInit(v reflect.Value) {
	if safeIsNil(v) && v.CanSet() {
		newField := reflect.New(f.elemType).Elem()
		for i := 0; i < f.ptrDeep; i++ {
			if newField.CanAddr() {
				newField = newField.Addr()
			} else {
				newField2 := reflect.New(newField.Type())
				newField2.Elem().Set(newField)
				newField = newField2
			}
		}
		v.Set(newField)
	}
}

func (s *structVM) copySubFields(field *fieldVM, sub *structVM) {
	nameSpace := field.structField.Name
	offset := field.offset
	ptrDeep := field.ptrDeep
	for _, k := range sub.fieldSelectorList {
		v := sub.fields[k]
		valueGetter := v.valueGetter
		reflectValueGetter := v.reflectValueGetter
		f := &fieldVM{
			structField: v.structField,
			offset:      offset + v.offset,
			origin:      v.origin,
			exprs:       make(map[string]*Expr, len(v.exprs)),
		}

		var selector string
		for k, v := range v.exprs {
			selector = nameSpace + FieldSeparator + k
			f.exprs[selector] = v
			s.exprs[selector] = v
			s.exprSelectorList = append(s.exprSelectorList, selector)
		}

		if valueGetter != nil {
			if ptrDeep == 0 {
				f.valueGetter = func(ptr uintptr) interface{} {
					return valueGetter(ptr + field.offset)
				}
				f.reflectValueGetter = func(ptr uintptr, initZero bool) reflect.Value {
					return reflectValueGetter(ptr+field.offset, initZero)
				}
			} else {
				f.valueGetter = func(ptr uintptr) interface{} {
					newField := reflect.NewAt(field.structField.Type, unsafe.Pointer(ptr+field.offset))
					for i := 0; i < ptrDeep; i++ {
						newField = newField.Elem()
					}
					if newField.IsNil() {
						return nil
					}
					return valueGetter(uintptr(newField.Pointer()))
				}
				f.reflectValueGetter = func(ptr uintptr, initZero bool) reflect.Value {
					newField := reflect.NewAt(field.structField.Type, unsafe.Pointer(ptr+field.offset))
					if initZero {
						field.ensureInit(newField.Elem())
					}
					for i := 0; i < ptrDeep; i++ {
						newField = newField.Elem()
					}
					if !initZero && newField.IsNil() {
						return reflect.Value{}
					}
					return reflectValueGetter(uintptr(newField.Pointer()), initZero)
				}
			}
		}
		fieldSelector := nameSpace + FieldSeparator + k
		s.fields[fieldSelector] = f
		s.fieldSelectorList = append(s.fieldSelectorList, fieldSelector)
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

func (vm *VM) getStructType(t reflect.Type) (reflect.Type, error) {
	structType := t
	for structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("unsupport type: %s", t.String())
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

// Field returns the field handler specified by the selector.
func (t *TagExpr) Field(fieldSelector string) (fh *FieldHandler, found bool) {
	f, ok := t.s.fields[fieldSelector]
	if !ok {
		return nil, false
	}
	return newFieldHandler(t, fieldSelector, f), true
}

// RangeFields loop through each field.
// When fn returns false, interrupt traversal and return false.
func (t *TagExpr) RangeFields(fn func(*FieldHandler) bool) bool {
	if list := t.s.fieldSelectorList; len(list) > 0 {
		fields := t.s.fields
		for _, fieldSelector := range list {
			if !fn(newFieldHandler(t, fieldSelector, fields[fieldSelector])) {
				return false
			}
		}
	}
	return true
}

// Eval evaluate the value of the struct tag expression by the selector expression.
// NOTE:
//  format: fieldName, fieldName.exprName, fieldName1.fieldName2.exprName1
//  result types: float64, string, bool, nil
func (t *TagExpr) Eval(exprSelector string) interface{} {
	expr, ok := t.s.exprs[exprSelector]
	if !ok {
		// Compatible with single mode or the expression with the name @
		if strings.HasSuffix(exprSelector, ExprNameSeparator) {
			exprSelector = exprSelector[:len(exprSelector)-1]
			if strings.HasSuffix(exprSelector, ExprNameSeparator) {
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
func (t *TagExpr) Range(fn func(es ExprSelector, eval func() interface{}) bool) bool {
	if list := t.s.exprSelectorList; len(list) > 0 {
		exprs := t.s.exprs
		for _, exprSelector := range list {
			if !fn(ExprSelector(exprSelector), func() interface{} {
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

	if list := t.s.fieldsWithIndirectStructVM; len(list) > 0 {
		for _, f := range list {
			v := f.packElemFrom(t.ptr)

			if f.elemKind == reflect.Map {
				for _, key := range v.MapKeys() {
					if f.mapKeyStructVM != nil {
						ptr := tpack.From(derefValue(key)).Pointer()
						if ptr == 0 {
							continue
						}
						if !f.mapKeyStructVM.newTagExpr(ptr).Range(fn) {
							return false
						}
					}
					if f.mapOrSliceElemStructVM != nil {
						ptr := tpack.From(derefValue(v.MapIndex(key))).Pointer()
						if ptr == 0 {
							continue
						}
						if !f.mapOrSliceElemStructVM.newTagExpr(ptr).Range(fn) {
							return false
						}
					}
				}

				// go version>=1.12

				//iter := v.MapRange()
				//for iter.Next() {
				//	if f.mapKeyStructVM != nil {
				//		ptr := tpack.From(derefValue(iter.Key())).Pointer()
				//		if ptr == 0 {
				//			continue
				//		}
				//		if !f.mapKeyStructVM.newTagExpr(ptr).Range(fn) {
				//			return false
				//		}
				//	}
				//	if f.mapOrSliceElemStructVM != nil {
				//		ptr := tpack.From(derefValue(iter.Value())).Pointer()
				//		if ptr == 0 {
				//			continue
				//		}
				//		if !f.mapOrSliceElemStructVM.newTagExpr(ptr).Range(fn) {
				//			return false
				//		}
				//	}
				//}

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
	idx := strings.LastIndex(selector, ExprNameSeparator)
	if idx != -1 {
		selector = selector[:idx]
	}
	idx = strings.LastIndex(selector, FieldSeparator)
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
