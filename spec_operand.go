package tagexpr

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// --------------------------- Operand ---------------------------

type groupExprNode struct {
	exprBackground
	boolPrefix bool
}

func newGroupExprNode() ExprNode { return &groupExprNode{boolPrefix: true} }

func readGroupExprNode(expr *string) (grp ExprNode, subExprNode *string) {
	s := *expr
	*expr = strings.TrimLeft(*expr, "!")
	i := len(s) - len(*expr)
	sptr := readPairedSymbol(expr, '(', ')')
	if sptr == nil {
		*expr = s
		return nil, nil
	}
	e := &groupExprNode{}
	var boolPrefix = true
	for ; i > 0; i-- {
		boolPrefix = !boolPrefix
	}
	e.boolPrefix = boolPrefix
	return e, sptr
}

func (ge *groupExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	if ge.rightOperand == nil {
		return nil
	}
	v := ge.rightOperand.Run(currField, tagExpr)
	if r, ok := v.(bool); ok {
		return ge.boolPrefix == r
	}
	return v
}

type boolExprNode struct {
	exprBackground
	val bool
}

var boolRegexp = regexp.MustCompile(`^!*(true|false)([\|&!= \t]{1}|$)`)

func readBoolExprNode(expr *string) ExprNode {
	s := boolRegexp.FindString(*expr)
	if s == "" {
		return nil
	}
	last := s[len(s)-1]
	if last != 'e' {
		s = s[:len(s)-1]
	}
	*expr = (*expr)[len(s):]
	e := &boolExprNode{}
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

func (be *boolExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return be.val }

type stringExprNode struct {
	exprBackground
	val string
}

func readStringExprNode(expr *string) ExprNode {
	sptr := readPairedSymbol(expr, '\'', '\'')
	if sptr == nil {
		return nil
	}
	e := &stringExprNode{val: *sptr}
	return e
}

func (se *stringExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return se.val }

type digitalExprNode struct {
	exprBackground
	val float64
}

var digitalRegexp = regexp.MustCompile(`^[\+\-]?\d+(\.\d+)?([\+\-\*\/%><\|&!=\^ \t\\]|$)`)

func readDigitalExprNode(expr *string) ExprNode {
	s := digitalRegexp.FindString(*expr)
	if s == "" {
		return nil
	}
	last := s[len(s)-1]
	if last < '0' || last > '9' {
		s = s[:len(s)-1]
	}
	*expr = (*expr)[len(s):]
	e := &digitalExprNode{}
	e.val, _ = strconv.ParseFloat(s, 64)
	return e
}

func (de *digitalExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return de.val }

func trimLeftSpace(p *string) *string {
	*p = strings.TrimLeftFunc(*p, unicode.IsSpace)
	return p
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
