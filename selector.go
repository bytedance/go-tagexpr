package tagexpr

import (
	"strings"
)

const (
	// ExprNameSeparator in the expression selector,
	// the separator of the field name and expression name
	ExprNameSeparator = "@"
	// DefaultExprName the default name of single model expression
	DefaultExprName = ExprNameSeparator
)

// ExprSelector expression selector
type ExprSelector string

// String returns string type value.
func (e ExprSelector) String() string {
	return string(e)
}

// Name returns the name of the expression.
func (e ExprSelector) Name() string {
	s := string(e)
	atIdx := strings.LastIndex(s, ExprNameSeparator)
	if atIdx == -1 {
		return DefaultExprName
	}
	return s[atIdx+1:]
}

// Field returns the name of the field it belongs to.
func (e ExprSelector) Field() string {
	s := string(e)
	idx := strings.LastIndex(s, ExprNameSeparator)
	if idx != -1 {
		s = s[:idx]
	}
	idx = strings.LastIndex(s, ".")
	if idx != -1 {
		return s[idx+1:]
	}
	return s
}
