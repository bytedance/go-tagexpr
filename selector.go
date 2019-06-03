package tagexpr

import (
	"strings"
)

// ExprSelector expression selector
type ExprSelector string

// String returns string type value.
func (e ExprSelector) String() string {
	return string(e)
}

// Name returns the name of expression.
func (e ExprSelector) Name() string {
	s := string(e)
	atIdx := strings.LastIndex(s, "@")
	if atIdx == -1 {
		return "@"
	}
	return s[atIdx+1:]
}
