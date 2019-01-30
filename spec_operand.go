// Copyright 2019 Bytedance Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	boolPrefix *bool
}

func newGroupExprNode() ExprNode { return &groupExprNode{} }

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
	if i > 0 {
		var bol = true
		for ; i > 0; i-- {
			bol = !bol
		}
		e.boolPrefix = &bol
	}
	return e, sptr
}

func (ge *groupExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	if ge.rightOperand == nil {
		return nil
	}
	v := ge.rightOperand.Run(currField, tagExpr)
	if ge.boolPrefix == nil {
		return v
	}
	if r, ok := v.(bool); ok {
		return *ge.boolPrefix == r
	}
	return nil
}

type boolExprNode struct {
	exprBackground
	val bool
}

var boolRegexp = regexp.MustCompile(`^!*(true|false)([\)\],\|&!= \t]{1}|$)`)

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

var digitalRegexp = regexp.MustCompile(`^[\+\-]?\d+(\.\d+)?([\)\],\+\-\*\/%><\|&!=\^ \t\\]|$)`)

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

type nilExprNode struct {
	exprBackground
}

var nilRegexp = regexp.MustCompile(`^nil([\)\],\|&!= \t]{1}|$)`)

func readNilExprNode(expr *string) ExprNode {
	s := nilRegexp.FindString(*expr)
	if s == "" {
		return nil
	}
	*expr = (*expr)[3:]
	e := &nilExprNode{}
	return e
}

func (*nilExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return nil }
