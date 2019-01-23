package tagexpr

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// TagExpr struct tag expression evaluator
type TagExpr interface {
	// Eval evaluate the value of the struct tag expression by the selector expression.
	// format: fieldName.exprName, fieldName1.fieldName2.exprName1
	Eval(selector string) interface{}
	// Range loop through each tag expression
	Range(func(selector string, eval func() interface{}))
}

// VM struct tag expression interpreter
type VM struct {
	tagName   string
	structJar map[string]*Struct
	rw        sync.RWMutex
}

// Struct tag expression set of struct
type Struct struct {
	name    string
	fields  map[string]*Field
	exprSet Set
}

// Field tag expression set of struct field
type Field struct {
	name   string
	host   *Struct
	sub    *Struct // struct field
	offset uintptr
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
	t, err := vm.getStructType(reflect.TypeOf(structOrStructPtr))
	if err == nil {
		vm.rw.Lock()
		defer vm.rw.Unlock()
		_, err = vm.warmUpLocked(t)
	}
	return err
}

func (vm *VM) warmUpLocked(structType reflect.Type) (*Struct, error) {
	err := vm.registerStruct(structType)
	if err != nil {
		return nil, err
	}
	return vm.structJar[structType.String()], nil
}

func (vm *VM) Run(structOrStructPtr interface{}) (TagExpr, error) {
	if structOrStructPtr == nil {
		return nil, errors.New("cannot run nil interface")
	}
	v, err := vm.getStructValue(reflect.ValueOf(structOrStructPtr))
	if err != nil {
		return nil, err
	}
	t := v.Type()

	vm.rw.RLock()
	s, ok := vm.structJar[t.String()]
	vm.rw.RUnlock()
	if !ok {
		vm.rw.Lock()
		s, ok = vm.structJar[t.String()]
		if !ok {
			s, err = vm.warmUpLocked(t)
			if err != nil {
				vm.rw.Unlock()
				return nil, err
			}
		}
		vm.rw.Unlock()
	}
	_ = s
	return nil, nil
}

func (vm *VM) registerStruct(structType reflect.Type) error {

	return nil
}

func (vm *VM) getStructValue(v reflect.Value) (reflect.Value, error) {
	structValue := v
	for structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}
	if structValue.Kind() != reflect.Struct {
		return v, fmt.Errorf("not structure pointer or structure: %s", v.Type().String())
	}
	return structValue, nil
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

// Set tag expression set
// key format: fieldName.exprName, fieldName1.fieldName2.exprName1
type Set map[string]*Expr

// Eval evaluate the value of the struct tag expression by the selector expression.
// format: fieldName.exprName, fieldName1.fieldName2.exprName1
func (t Set) Eval(selector string) interface{} {
	return nil
}

// Range loop through each tag expression
func (t Set) Range(func(selector string, eval func() interface{})) {

}
