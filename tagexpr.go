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
	sub         *Struct // struct field
	valueGetter func(uintptr) interface{}
}

// New creates a tag expression interpreter that uses @tagName as the tag name.
func New(tagName string) *VM {
	return &VM{
		tagName:   tagName,
		structJar: make(map[string]*Struct, 256),
	}
}

func (vm *VM) WarmUp(structOrStructPtr interface{}) error {
	if structOrStructPtr == nil {
		return errors.New("cannot warn up nil interface")
	}
	vm.rw.Lock()
	defer vm.rw.Unlock()
	return vm.registerStructLocked(reflect.TypeOf(structOrStructPtr))
}

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
			err = vm.registerStructLocked(t)
			if err != nil {
				vm.rw.Unlock()
				return nil, err
			}
			s = vm.structJar[tname]
		}
		vm.rw.Unlock()
	}
	return s.newTagExpr(v.Pointer()), nil
}

func (vm *VM) registerStructLocked(structType reflect.Type) error {
	structType, err := vm.getStructType(structType)
	if err != nil {
		return err
	}
	structTypeName := structType.String()
	s, had := vm.structJar[structTypeName]
	if had {
		return nil
	}
	s = vm.newStruct()
	vm.structJar[structTypeName] = s
	var numField = structType.NumField()
	var structField reflect.StructField
	for i := 0; i < numField; i++ {
		structField = structType.Field(i)
		field, err := s.newField(structField)
		if err != nil {
			return err
		}
		switch structField.Type.Kind() {
		default:
			field.valueGetter = func(ptr uintptr) interface{} { return nil }
		case reflect.Struct:
			vm.registerStructLocked(field.Type)
		case reflect.Float32, reflect.Float64:
			field.setFloatGetter()
		case reflect.String:
			field.setStringGetter()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.setIntGetter()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.setUintGetter()
		}
	}
	return nil
}

func (f *Field) newFrom(ptr uintptr) reflect.Value {
	fieldPtr := unsafe.Pointer(ptr + f.Offset)
	return reflect.NewAt(f.Type, fieldPtr).Elem()
}

func (f *Field) setFloatGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr).Float()
	}
}

func (f *Field) setIntGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		return float64(f.newFrom(ptr).Int())
	}
}

func (f *Field) setUintGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		return float64(f.newFrom(ptr).Uint())
	}
}

func (f *Field) setBoolGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr).Bool()
	}
}

func (f *Field) setStringGetter() {
	f.valueGetter = func(ptr uintptr) interface{} {
		return f.newFrom(ptr).String()
	}
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
		f.host.exprs[f.Name] = expr
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
	expr, ok := t.s.exprs[selector]
	if !ok {
		return nil
	}
	return expr.run(getFieldSelector(selector), t)
}

func getFieldSelector(selector string) string {
	idx := strings.LastIndex(selector, ".")
	if idx == -1 {
		return selector
	}
	return selector[:idx]
}

// Range loop through each tag expression
func (t *TagExpr) Range(fn func(selector string, eval func() interface{})) {
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
	_ = subFields // TODO
	return f.valueGetter(t.ptr)
}
