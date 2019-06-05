package tagexpr

import "reflect"

// FieldHandler field handler
type FieldHandler struct {
	selector string
	field    *fieldVM
	expr     *TagExpr
}

func newFieldHandler(expr *TagExpr, fieldSelector string, field *fieldVM) *FieldHandler {
	return &FieldHandler{
		selector: fieldSelector,
		field:    field,
		expr:     expr,
	}
}

// StringSelector returns the field selector of string type.
func (f *FieldHandler) StringSelector() string {
	return f.selector
}

// FieldSelector returns the field selector of FieldSelector type.
func (f *FieldHandler) FieldSelector() FieldSelector {
	return FieldSelector(f.selector)
}

// Value returns the field value.
// NOTE:
//  If initZero==true, initialize nil pointer to zero value
func (f *FieldHandler) Value(initZero bool) reflect.Value {
	return f.field.reflectValueGetter(f.expr.ptr, initZero)
}

// EvalFuncs returns the tag expression eval functions.
func (f *FieldHandler) EvalFuncs() map[ExprSelector]func() interface{} {
	targetTagExpr, _ := f.expr.checkout(f.selector)
	evals := make(map[ExprSelector]func() interface{}, len(f.field.exprs))
	for k, v := range f.field.exprs {
		expr := v
		exprSelector := ExprSelector(k)
		evals[exprSelector] = func() interface{} {
			return expr.run(exprSelector.Name(), targetTagExpr)
		}
	}
	return evals
}

// StructField returns the field StructField object.
func (f *FieldHandler) StructField() reflect.StructField {
	return f.field.structField
}
