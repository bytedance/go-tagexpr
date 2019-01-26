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
	"strings"
)

type selectorExprNode struct {
	exprBackground
	field, name string
	subExprs    []ExprNode
	boolPrefix  *bool
}

func (p *Expr) readSelectorExprNode(expr *string) ExprNode {
	field, name, subSelector, boolPrefix, found := findSelector(expr)
	if !found {
		return nil
	}
	operand := &selectorExprNode{
		field:      field,
		name:       name,
		boolPrefix: boolPrefix,
	}
	operand.subExprs = make([]ExprNode, 0, len(subSelector))
	for _, s := range subSelector {
		grp := newGroupExprNode()
		_, err := p.parseExprNode(&s, grp)
		if err != nil {
			return nil
		}
		sortPriority(grp.RightOperand())
		operand.subExprs = append(operand.subExprs, grp)
	}
	return operand
}

var selectorRegexp = regexp.MustCompile(`^(\!*)(\([ \t]*[A-Za-z_]+[A-Za-z0-9_\.]*[ \t]*\))?(\$)([\[\+\-\*\/%><\|&!=\^ \t\\]|$)`)

func findSelector(expr *string) (field string, name string, subSelector []string, boolPrefix *bool, found bool) {
	raw := *expr
	a := selectorRegexp.FindAllStringSubmatch(raw, -1)
	if len(a) != 1 {
		return
	}
	r := a[0]
	if s0 := r[2]; len(s0) > 0 {
		field = strings.TrimSpace(s0[1 : len(s0)-1])
	}
	name = r[3]
	*expr = (*expr)[len(a[0][0])-len(r[4]):]
	for {
		sub := readPairedSymbol(expr, '[', ']')
		if sub == nil {
			break
		}
		if *sub == "" || (*sub)[0] == '[' {
			*expr = raw
			return "", "", nil, nil, false
		}
		subSelector = append(subSelector, strings.TrimSpace(*sub))
	}
	if boolNum := len(r[1]); boolNum > 0 {
		bol := true
		for i := len(r[1]); i > 0; i-- {
			bol = !bol
		}
		boolPrefix = &bol
	}
	found = true
	return
}

func (ve *selectorExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	var subFields []interface{}
	if n := len(ve.subExprs); n > 0 {
		subFields = make([]interface{}, n)
		for i, e := range ve.subExprs {
			subFields[i] = e.Run(currField, tagExpr)
		}
	}
	field := ve.field
	if field == "" {
		field = currField
	}
	v := tagExpr.getValue(field, subFields)
	if ve.boolPrefix == nil {
		return v
	}
	if r, ok := v.(bool); ok {
		return *ve.boolPrefix == r
	}
	return nil
}
