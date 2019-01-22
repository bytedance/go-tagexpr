package tagexpr

import (
	"math"
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

// 	Greater                           // >
// 	GreaterEqual                      // >=
// 	Less                              // <
// 	LessEqual                         // <=
// 	And                               // &&
// 	Or                                // ||

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

type groupExpr struct {
	exprBackground
	boolPrefix bool
}

func newGroupExpr() Expr { return &groupExpr{boolPrefix: true} }

func readGroupExpr(expr *string) (grp Expr, subExpr *string) {
	s := *expr
	*expr = strings.TrimLeft(*expr, "!")
	i := len(s) - len(*expr)
	sptr := readPairedSymbol(expr, '(', ')')
	if sptr == nil {
		*expr = s
		return nil, nil
	}
	e := &groupExpr{}
	var boolPrefix = true
	for ; i > 0; i-- {
		boolPrefix = !boolPrefix
	}
	e.boolPrefix = boolPrefix
	return e, sptr
}

func (ge *groupExpr) Calculate() interface{} {
	v := ge.rightOperand.Calculate()
	if r, ok := v.(bool); ok {
		return ge.boolPrefix == r
	}
	return v
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

var digitalRegexp = regexp.MustCompile(`^[\+\-]?\d+(\.\d+)?([\+\-\*\/%><\|&!=\^ \t\\]|$)`)

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

type divisionExpr struct{ exprBackground }

func newDivisionExpr() Expr { return &divisionExpr{} }

func (de *divisionExpr) Calculate() interface{} {
	v1, _ := de.rightOperand.Calculate().(float64)
	if v1 == 0 {
		return math.NaN()
	}
	v0, _ := de.leftOperand.Calculate().(float64)
	return v0 / v1
}

type subtractionExpr struct{ exprBackground }

func newSubtractionExpr() Expr { return &subtractionExpr{} }

func (de *subtractionExpr) Calculate() interface{} {
	v0, _ := de.leftOperand.Calculate().(float64)
	v1, _ := de.rightOperand.Calculate().(float64)
	return v0 - v1
}

type remainderExpr struct{ exprBackground }

func newRemainderExpr() Expr { return &remainderExpr{} }

func (re *remainderExpr) Calculate() interface{} {
	v1, _ := re.rightOperand.Calculate().(float64)
	if v1 == 0 {
		return math.NaN()
	}
	v0, _ := re.leftOperand.Calculate().(float64)
	return float64(int64(v0) % int64(v1))
}

type equalExpr struct{ exprBackground }

func newEqualExpr() Expr { return &equalExpr{} }

func (ee *equalExpr) Calculate() interface{} {
	v0 := ee.leftOperand.Calculate()
	v1 := ee.rightOperand.Calculate()
	switch r := v0.(type) {
	case float64:
		var r1 float64
		r1, _ = v1.(float64)
		return r == r1
	case string:
		var r1 string
		r1, _ = v1.(string)
		return r == r1
	case bool:
		var r1 bool
		r1, _ = v1.(bool)
		return r == r1
	default:
		return false
	}
}

type notEqualExpr struct{ equalExpr }

func newNotEqualExpr() Expr { return &notEqualExpr{} }

func (ne *notEqualExpr) Calculate() interface{} {
	return !ne.equalExpr.Calculate().(bool)
}

type greaterExpr struct{ exprBackground }

func newGreaterExpr() Expr { return &greaterExpr{} }

func (ge *greaterExpr) Calculate() interface{} {
	v0 := ge.leftOperand.Calculate()
	v1 := ge.rightOperand.Calculate()
	switch r := v0.(type) {
	case float64:
		var r1 float64
		r1, _ = v1.(float64)
		return r > r1
	case string:
		var r1 string
		r1, _ = v1.(string)
		return r > r1
	default:
		return false
	}
}

type greaterEqualExpr struct{ exprBackground }

func newGreaterEqualExpr() Expr { return &greaterEqualExpr{} }

func (ge *greaterEqualExpr) Calculate() interface{} {
	v0 := ge.leftOperand.Calculate()
	v1 := ge.rightOperand.Calculate()
	switch r := v0.(type) {
	case float64:
		var r1 float64
		r1, _ = v1.(float64)
		return r >= r1
	case string:
		var r1 string
		r1, _ = v1.(string)
		return r >= r1
	default:
		return false
	}
}

type lessExpr struct{ exprBackground }

func newLessExpr() Expr { return &lessExpr{} }

func (le *lessExpr) Calculate() interface{} {
	v0 := le.leftOperand.Calculate()
	v1 := le.rightOperand.Calculate()
	switch r := v0.(type) {
	case float64:
		var r1 float64
		r1, _ = v1.(float64)
		return r < r1
	case string:
		var r1 string
		r1, _ = v1.(string)
		return r < r1
	default:
		return false
	}
}

type lessEqualExpr struct{ exprBackground }

func newLessEqualExpr() Expr { return &lessEqualExpr{} }

func (le *lessEqualExpr) Calculate() interface{} {
	v0 := le.leftOperand.Calculate()
	v1 := le.rightOperand.Calculate()
	switch r := v0.(type) {
	case float64:
		var r1 float64
		r1, _ = v1.(float64)
		return r <= r1
	case string:
		var r1 string
		r1, _ = v1.(string)
		return r <= r1
	default:
		return false
	}
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
