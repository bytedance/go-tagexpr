package tagexpr

import (
	"regexp"
	"strconv"
	"strings"
)

// Expr expression interface
type Expr interface {
	SetParent(Expr)
	Parent() Expr
	SetLeftOperand(Expr)
	SetRightOperand(Expr)
	Calculate() interface{}
}

// // Operator kind
// type Operator uint

// // Operator enumerate
// const (
// 	Addition          Operator = iota // \+
// 	Subtraction                       // -
// 	Multiplication                    // \*
// 	Division                          // \/
// 	Remainder                         // %
// 	BitwiseAnd                        // &
// 	BitwiseOr                         // \|
// 	BitwiseXor                        // \^
// 	BitwiseClean                      // &\^
// 	BitwiseShiftLeft                  // <<
// 	BitwiseShiftRight                 // >>
// 	Equal                             // ==
// 	NotEqual                          // !=
// 	Greater                           // >
// 	GreaterEqual                      // >=
// 	Less                              // <
// 	LessEqual                         // <=
// 	And                               // &&
// 	Or                                // \|\|
// 	Group                             // \(\)
// 	String                            // '\w'
// 	Digital                           // \d+(\.\d+)?
// )

type exprBackground struct {
	parent       Expr
	leftOperand  Expr
	rightOperand Expr
}

func (eb *exprBackground) SetParent(e Expr) {
	eb.parent = e
}

func (eb *exprBackground) Parent() Expr {
	return eb.parent
}

func (eb *exprBackground) SetLeftOperand(left Expr) {
	eb.leftOperand = left
}

func (eb *exprBackground) SetRightOperand(right Expr) {
	eb.rightOperand = right
}

func (*exprBackground) Calculate() interface{} { return nil }

type groupExpr struct{ exprBackground }

func newGroupExpr() Expr { return &groupExpr{} }

func readGroupExpr(expr *string) (grp Expr, subExpr *string) {
	sptr := readPairedSymbol(expr, '(', ')')
	if sptr == nil {
		return nil, nil
	}
	e := &groupExpr{}
	return e, sptr
}

func (ge *groupExpr) Calculate() interface{} {
	return ge.rightOperand.Calculate()
}

type boolExpr struct {
	exprBackground
	val bool
}

var boolRegexp = regexp.MustCompile(`^!*(true|false)([\|&!= \t]{1}|$)`)

func readBoolExpr(expr *string) Expr {
	s := boolRegexp.FindString(*expr)
	if s == "" {
		return nil
	}
	last := s[len(s)-1]
	if last != 'e' {
		s = s[:len(s)-1]
	}
	*expr = (*expr)[len(s):]
	e := &boolExpr{}
	if strings.Contains(s, "t") {
		var v = true
		for i := len(s) - 4; i > 0; i-- {
			v = !v
		}
		e.val = v
	} else {
		var v = false
		for i := len(s) - 5; i > 0; i-- {
			v = !v
		}
		e.val = v
	}
	return e
}

func (be *boolExpr) Calculate() interface{} { return be.val }

type stringExpr struct {
	exprBackground
	val string
}

func readStringExpr(expr *string) Expr {
	sptr := readPairedSymbol(expr, '\'', '\'')
	if sptr == nil {
		return nil
	}
	e := &stringExpr{val: *sptr}
	return e
}

func (se *stringExpr) Calculate() interface{} { return se.val }

type digitalExpr struct {
	exprBackground
	val float64
}

var digitalRegexp = regexp.MustCompile(`^[\+\-]?\d+(\.\d+)?([\+\-\|&%!=\*\^ \t\\]|$)`)

func readDigitalExpr(expr *string) Expr {
	s := digitalRegexp.FindString(*expr)
	if s == "" {
		return nil
	}
	last := s[len(s)-1]
	if last < '0' || last > '9' {
		s = s[:len(s)-1]
	}
	*expr = (*expr)[len(s):]
	e := &digitalExpr{}
	e.val, _ = strconv.ParseFloat(s, 64)
	return e
}

func (de *digitalExpr) Calculate() interface{} { return de.val }

type additionExpr struct{ exprBackground }

func newAdditionExpr() Expr { return &additionExpr{} }

func (ae *additionExpr) Calculate() interface{} {
	// positive number or Addition
	v0 := ae.leftOperand.Calculate()
	v1 := ae.rightOperand.Calculate()
	switch r := v0.(type) {
	case float64:
		var v float64
		v, _ = v1.(float64)
		r += v
		return r
	case string:
		var v string
		v, _ = v1.(string)
		r += v
		return r
	default:
		return v1
	}
}

type multiplicationExpr struct{ exprBackground }

func newMultiplicationExpr() Expr { return &multiplicationExpr{} }

func (ae *multiplicationExpr) Calculate() interface{} {
	v0, _ := ae.leftOperand.Calculate().(float64)
	v1, _ := ae.rightOperand.Calculate().(float64)
	return v0 * v1
}

func readPairedSymbol(p *string, left, right rune) *string {
	s := *p
	if len(s) == 0 || rune(s[0]) != left {
		return nil
	}
	s = s[1:]
	var last1 = left
	var last2 rune
	var leftLevel, rightLevel int
	for i, r := range s {
		if r == right && (last1 != '\\' || last2 == '\\') {
			if leftLevel == rightLevel {
				*p = s[i+1:]
				sub := s[:i]
				return &sub
			}
			rightLevel++
		} else if r == left && (last1 != '\\' || last2 == '\\') {
			leftLevel++
		}
		last2 = last1
		last1 = r
	}
	return nil
}
