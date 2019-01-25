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
	structJar map[string]*Struct
	rw        sync.RWMutex
}

// Struct tag expression set of struct
type Struct struct {
	vm     *VM
	name   string
	fields map[string]*Field
	exprs  map[string]*Expr
}

// Field tag expression set of struct field
type Field struct {
	reflect.StructField
	host        *Struct
	valueGetter func(uintptr) interface{}
}

// New creates a tag expression interpreter that uses @tagName as the tag name.
func New(tagName string) *VM {
	return &VM{
		tagName:   tagName,
		structJar: make(map[string]*Struct, 256),
	}
}

// WarmUp warms up the interpreter of the struct type to
// improve the performance of the vm.Run .
func (vm *VM) WarmUp(structOrStructPtr interface{}) error {
	if structOrStructPtr == nil {
		return errors.New("cannot warn up nil interface")
	}
	vm.rw.Lock()
	defer vm.rw.Unlock()
	_, err := vm.registerStructLocked(reflect.TypeOf(structOrStructPtr))
	return err
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

func (vm *VM) registerStructLocked(structType reflect.Type) (*Struct, error) {
	structType, err := vm.getStructType(structType)
	if err != nil {
		return nil, err
	}
	structTypeName := structType.String()
	s, had := vm.structJar[structTypeName]
	if had {
		return s, nil
	}
	s = vm.newStruct()
	vm.structJar[structTypeName] = s
	var numField = structType.NumField()
	var structField reflect.StructField
	var sub *Struct
	for i := 0; i < numField; i++ {
		structField = structType.Field(i)
		field, err := s.newField(structField)
		if err != nil {
			return nil, err
		}
		t := structField.Type
		var ptrDeep int
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
			ptrDeep++
		}
		switch t.Kind() {
		default:
			field.valueGetter = func(ptr uintptr) interface{} { return nil }
		case reflect.Struct:
			sub, err = vm.registerStructLocked(field.Type)
			if err != nil {
				return nil, err
			}
			s.copySubFields(field, sub, ptrDeep)
		case reflect.Float32, reflect.Float64:
			field.setFloatGetter(ptrDeep)
		case reflect.String:
			field.setStringGetter(ptrDeep)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.setIntGetter(ptrDeep)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.setUintGetter(ptrDeep)
		case reflect.Bool:
			field.setBoolGetter(ptrDeep)
		case reflect.Map, reflect.Array, reflect.Slice:
			field.setLengthGetter(ptrDeep)
		}
	}
	return s, nil
}

func (vm *VM) newStruct() *Struct {
	return &Struct{
		vm:     vm,
		fields: make(map[string]*Field, 10),
		exprs:  make(map[string]*Expr, 40),
	}
}

func (s *Struct) newField(structField reflect.StructField) (*Field, error) {
	f := &Field{
		StructField: structField,
		host:        s,
	}
	err := f.parseExprs(structField.Tag.Get(s.vm.tagName))
	if err != nil {
		return nil, err
	}
	s.fields[f.Name] = f
	return f, nil
}

func (f *Field) newFrom(ptr uintptr, ptrDeep int) reflect.Value {
	v := reflect.NewAt(f.Type, unsafe.Pointer(ptr+f.Offset)).Elem()
	for i := 0; i < ptrDeep; i++ {
		v = v.Elem()
	}
	return v
}

func (f *Field) setFloatGetter(ptrDeep int) {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr, ptrDeep).Float()
	}
}

func (f *Field) setIntGetter(ptrDeep int) {
	f.valueGetter = func(ptr uintptr) interface{} {
		return float64(f.newFrom(ptr, ptrDeep).Int())
	}
}

func (f *Field) setUintGetter(ptrDeep int) {
	f.valueGetter = func(ptr uintptr) interface{} {
		return float64(f.newFrom(ptr, ptrDeep).Uint())
	}
}

func (f *Field) setBoolGetter(ptrDeep int) {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr, ptrDeep).Bool()
	}
}

func (f *Field) setStringGetter(ptrDeep int) {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr, ptrDeep).String()
	}
}

func (f *Field) setLengthGetter(ptrDeep int) {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr, ptrDeep).Interface()
	}
}

func (f *Field) parseExprs(tag string) error {
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
		f.host.exprs[f.Name+".$"] = expr
		return nil
	}
	var subtag *string
	var idx int
	var exprName, exprStr string
	for {
		subtag = readPairedSymbol(&tag, '{', '}')
		if subtag != nil {
			idx = strings.Index(*subtag, ":")
			if idx > 0 {
				exprName = strings.TrimSpace((*subtag)[:idx])
				if exprName != "" {
					exprName = f.Name + "." + exprName
					if _, had := f.host.exprs[exprName]; had {
						return fmt.Errorf("duplicate expression name: %s", exprName)
					}
					exprStr = strings.TrimSpace((*subtag)[idx+1:])
					if exprStr != "" {
						if expr, err := parseExpr(exprStr); err == nil {
							f.host.exprs[exprName] = expr
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
		}
		return fmt.Errorf("syntax incorrect: %q", raw)
	}
}

func (s *Struct) copySubFields(field *Field, sub *Struct, ptrDeep int) {
	nameSpace := field.Name
	for k, v := range sub.fields {
		valueGetter := v.valueGetter
		f := &Field{
			StructField: v.StructField,
			host:        v.host,
		}
		if valueGetter != nil {
			if ptrDeep == 0 {
				f.valueGetter = func(ptr uintptr) interface{} {
					return valueGetter(ptr + field.Offset)
				}
			} else {
				f.valueGetter = func(ptr uintptr) interface{} {
					newField := reflect.NewAt(field.Type, unsafe.Pointer(ptr+field.Offset))
					for i := 0; i < ptrDeep; i++ {
						newField = newField.Elem()
					}
					return valueGetter(uintptr(newField.Pointer()))
				}
			}
		}
		s.fields[nameSpace+"."+k] = f
	}
	for k, v := range sub.exprs {
		s.exprs[nameSpace+"."+k] = v
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

func (s *Struct) newTagExpr(ptr uintptr) *TagExpr {
	te := &TagExpr{
		s:   s,
		ptr: ptr,
	}
	return te
}

// TagExpr struct tag expression evaluator
type TagExpr struct {
	s   *Struct
	ptr uintptr
}

// Eval evaluate the value of the struct tag expression by the selector expression.
// format: fieldName, fieldName.exprName, fieldName1.fieldName2.exprName1
func (t *TagExpr) Eval(selector string) interface{} {
	defer func() {
		if recover() != nil {
			// fmt.Println(goutil.BytesToString(goutil.PanicTrace(1)))
		}
	}()
	expr, ok := t.s.exprs[selector]
	if !ok {
		return nil
	}
	return expr.run(getFieldSelector(selector), t)
}

// Range loop through each tag expression
func (t *TagExpr) Range(fn func(selector string, eval func() interface{})) {
	defer func() {
		if recover() != nil {
			// fmt.Println(goutil.BytesToString(goutil.PanicTrace(1)))
		}
	}()
	for selector, expr := range t.s.exprs {
		fn(selector, func() interface{} {
			return expr.run(getFieldSelector(selector), t)
		})
	}
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
	if len(subFields) == 0 {
		return v
	}
	vv := reflect.ValueOf(v)
	for _, k := range subFields {
		for vv.Kind() == reflect.Ptr {
			vv = vv.Elem()
		}
		switch vv.Kind() {
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
			vv = vv.MapIndex(reflect.ValueOf(k).Convert(vv.Type().Key()))
		default:
			return nil
		}
	}
	for vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}
	switch vv.Kind() {
	default:
		if vv.CanInterface() {
			return vv.Interface()
		}
		return nil
	case reflect.String:
		return vv.String()
	case reflect.Bool:
		return vv.Bool()
	case reflect.Float32, reflect.Float64:
		return vv.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vv.Convert(float64Type).Float()
	}
}

var float64Type = reflect.TypeOf(float64(0))

func getFieldSelector(selector string) string {
	idx := strings.LastIndex(selector, ".")
	if idx == -1 {
		return selector
	}
	return selector[:idx]
}
