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
	boolOpposite *bool
}

func newGroupExprNode() ExprNode { return &groupExprNode{} }

func readGroupExprNode(expr *string) (grp ExprNode, subExprNode *string) {
	last, boolOpposite := getBoolOpposite(expr)
	sptr := readPairedSymbol(&last, '(', ')')
	if sptr == nil {
		return nil, nil
	}
	*expr = last
	e := &groupExprNode{boolOpposite: boolOpposite}
	return e, sptr
}

func (ge *groupExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	if ge.rightOperand == nil {
		return nil
	}
	return realValue(ge.rightOperand.Run(currField, tagExpr), ge.boolOpposite)
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
		e.val = (len(s)-4)%2 == 0
	} else {
		e.val = (len(s)-5)%2 == 1
	}
	return e
}

func (be *boolExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return be.val }

type stringExprNode struct {
	exprBackground
	val interface{}
}

func readStringExprNode(expr *string) ExprNode {
	last, boolOpposite := getBoolOpposite(expr)
	sptr := readPairedSymbol(&last, '\'', '\'')
	if sptr == nil {
		return nil
	}
	*expr = last
	e := &stringExprNode{val: realValue(*sptr, boolOpposite)}
	return e
}

func (se *stringExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return se.val }

type digitalExprNode struct {
	exprBackground
	val interface{}
}

var digitalRegexp = regexp.MustCompile(`^[\+\-]?\d+(\.\d+)?([\)\],\+\-\*\/%><\|&!=\^ \t\\]|$)`)

func readDigitalExprNode(expr *string) ExprNode {
	last, boolOpposite := getBoolOpposite(expr)
	s := digitalRegexp.FindString(last)
	if s == "" {
		return nil
	}
	if r := s[len(s)-1]; r < '0' || r > '9' {
		s = s[:len(s)-1]
	}
	*expr = last[len(s):]
	f64, _ := strconv.ParseFloat(s, 64)
	return &digitalExprNode{val: realValue(f64, boolOpposite)}
}

func (de *digitalExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return de.val }

type nilExprNode struct {
	exprBackground
	val interface{}
}

var nilRegexp = regexp.MustCompile(`^nil([\)\],\|&!= \t]{1}|$)`)

func readNilExprNode(expr *string) ExprNode {
	last, boolOpposite := getBoolOpposite(expr)
	s := nilRegexp.FindString(last)
	if s == "" {
		return nil
	}
	*expr = last[3:]
	return &nilExprNode{val: realValue(nil, boolOpposite)}
}

func (ne *nilExprNode) Run(currField string, tagExpr *TagExpr) interface{} { return ne.val }

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
	var escapeIndexes = make(map[int]bool)
	var realEqual, escapeEqual bool
	for i, r := range s {
		if realEqual, escapeEqual = equalRune(right, r, last1, last2); realEqual {
			if leftLevel == rightLevel {
				*p = s[i+1:]
				var sub = make([]rune, 0, i)
				for k, v := range s[:i] {
					if !escapeIndexes[k] {
						sub = append(sub, v)
					}
				}
				s = string(sub)
				return &s
			}
			rightLevel++
		} else if escapeEqual {
			escapeIndexes[i-1] = true
		} else if realEqual, escapeEqual = equalRune(left, r, last1, last2); realEqual {
			leftLevel++
		} else if escapeEqual {
			escapeIndexes[i-1] = true
		}
		last2 = last1
		last1 = r
	}
	return nil
}

func equalRune(a, b, last1, last2 rune) (real, escape bool) {
	if a == b {
		real = last1 != '\\' || last2 == '\\'
		escape = last1 == '\\' && last2 != '\\'
	}
	return
}

func getBoolOpposite(expr *string) (string, *bool) {
	last := strings.TrimLeft(*expr, "!")
	n := len(*expr) - len(last)
	if n == 0 {
		return last, nil
	}
	bol := n%2 == 1
	return last, &bol
}

func realValue(v interface{}, boolOpposite *bool) interface{} {
	if boolOpposite == nil {
		return v
	}
	bol := FakeBool(v)
	if *boolOpposite {
		return !bol
	}
	return bol
}
