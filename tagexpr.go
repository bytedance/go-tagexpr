package tagexpr

// TagExpr struct tag expression
type TagExpr struct {
}

// New creates a struct tag expression
func New(structPtr interface{}) *TagExpr {
	t := &TagExpr{}
	return t
}

// func
