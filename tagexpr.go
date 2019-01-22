package tagexpr

// TagExpr struct tag expression evaluator
type TagExpr interface {
	// Eval evaluate the value of the struct tag expression by the selector expression.
	// format: fieldName.exprName, fieldName1.fieldName2.exprName1
	Eval(selector string) interface{}
	// Range loop through each tag expression
	Range(func(selector string, eval func() interface{}))
}

// Struct struct tag expression
type Struct struct {
	name    string
	fields  map[string]*Field
	exprLib Lib
}

// Lib tag expression lib
// key format: fieldName.exprName, fieldName1.fieldName2.exprName1
type Lib map[string]*Expr

// Eval evaluate the value of the struct tag expression by the selector expression.
// format: fieldName.exprName, fieldName1.fieldName2.exprName1
func (t Lib) Eval(selector string) interface{} {
	return nil
}

// Range loop through each tag expression
func (t Lib) Range(func(selector string, eval func() interface{})) {

}

type Field struct {
	name   string
	host   *Struct
	sub    *Struct // struct field
	offset uintptr
}

// New creates a struct tag expression
func New(structPtr interface{}) *Struct {
	t := &Struct{}
	return t
}

func (s *Struct) Run(structPtr interface{}) TagExpr {
	return nil
}
