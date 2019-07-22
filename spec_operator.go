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
	"math"
)

// --------------------------- Operator ---------------------------

type additionExprNode struct{ exprBackground }

func newAdditionExprNode() ExprNode { return &additionExprNode{} }

func (ae *additionExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	// positive number or Addition
	v0 := ae.leftOperand.Run(currField, tagExpr)
	v1 := ae.rightOperand.Run(currField, tagExpr)
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

type multiplicationExprNode struct{ exprBackground }

func newMultiplicationExprNode() ExprNode { return &multiplicationExprNode{} }

func (ae *multiplicationExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0, _ := ae.leftOperand.Run(currField, tagExpr).(float64)
	v1, _ := ae.rightOperand.Run(currField, tagExpr).(float64)
	return v0 * v1
}

type divisionExprNode struct{ exprBackground }

func newDivisionExprNode() ExprNode { return &divisionExprNode{} }

func (de *divisionExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v1, _ := de.rightOperand.Run(currField, tagExpr).(float64)
	if v1 == 0 {
		return math.NaN()
	}
	v0, _ := de.leftOperand.Run(currField, tagExpr).(float64)
	return v0 / v1
}

type subtractionExprNode struct{ exprBackground }

func newSubtractionExprNode() ExprNode { return &subtractionExprNode{} }

func (de *subtractionExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0, _ := de.leftOperand.Run(currField, tagExpr).(float64)
	v1, _ := de.rightOperand.Run(currField, tagExpr).(float64)
	return v0 - v1
}

type remainderExprNode struct{ exprBackground }

func newRemainderExprNode() ExprNode { return &remainderExprNode{} }

func (re *remainderExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v1, _ := re.rightOperand.Run(currField, tagExpr).(float64)
	if v1 == 0 {
		return math.NaN()
	}
	v0, _ := re.leftOperand.Run(currField, tagExpr).(float64)
	return float64(int64(v0) % int64(v1))
}

type equalExprNode struct{ exprBackground }

func newEqualExprNode() ExprNode { return &equalExprNode{} }

func (ee *equalExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0 := ee.leftOperand.Run(currField, tagExpr)
	v1 := ee.rightOperand.Run(currField, tagExpr)
	switch r := v0.(type) {
	case float64:
		r1, ok := v1.(float64)
		if ok {
			return r == r1
		}
	case string:
		r1, ok := v1.(string)
		if ok {
			return r == r1
		}
	case bool:
		r1, ok := v1.(bool)
		if ok {
			return r == r1
		}
	case nil:
		return v1 == nil
	}
	return false
}

type notEqualExprNode struct{ equalExprNode }

func newNotEqualExprNode() ExprNode { return &notEqualExprNode{} }

func (ne *notEqualExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	return !ne.equalExprNode.Run(currField, tagExpr).(bool)
}

type greaterExprNode struct{ exprBackground }

func newGreaterExprNode() ExprNode { return &greaterExprNode{} }

func (ge *greaterExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0 := ge.leftOperand.Run(currField, tagExpr)
	v1 := ge.rightOperand.Run(currField, tagExpr)
	switch r := v0.(type) {
	case float64:
		r1, ok := v1.(float64)
		if ok {
			return r > r1
		}
	case string:
		r1, ok := v1.(string)
		if ok {
			return r > r1
		}
	}
	return false
}

type greaterEqualExprNode struct{ exprBackground }

func newGreaterEqualExprNode() ExprNode { return &greaterEqualExprNode{} }

func (ge *greaterEqualExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0 := ge.leftOperand.Run(currField, tagExpr)
	v1 := ge.rightOperand.Run(currField, tagExpr)
	switch r := v0.(type) {
	case float64:
		r1, ok := v1.(float64)
		if ok {
			return r >= r1
		}
	case string:
		r1, ok := v1.(string)
		if ok {
			return r >= r1
		}
	}
	return false
}

type lessExprNode struct{ exprBackground }

func newLessExprNode() ExprNode { return &lessExprNode{} }

func (le *lessExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0 := le.leftOperand.Run(currField, tagExpr)
	v1 := le.rightOperand.Run(currField, tagExpr)
	switch r := v0.(type) {
	case float64:
		r1, ok := v1.(float64)
		if ok {
			return r < r1
		}
	case string:
		r1, ok := v1.(string)
		if ok {
			return r < r1
		}
	}
	return false
}

type lessEqualExprNode struct{ exprBackground }

func newLessEqualExprNode() ExprNode { return &lessEqualExprNode{} }

func (le *lessEqualExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	v0 := le.leftOperand.Run(currField, tagExpr)
	v1 := le.rightOperand.Run(currField, tagExpr)
	switch r := v0.(type) {
	case float64:
		r1, ok := v1.(float64)
		if ok {
			return r <= r1
		}
	case string:
		r1, ok := v1.(string)
		if ok {
			return r <= r1
		}
	}
	return false
}

type andExprNode struct{ exprBackground }

func newAndExprNode() ExprNode { return &andExprNode{} }

func (ae *andExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	for _, e := range [2]ExprNode{ae.leftOperand, ae.rightOperand} {
		if !FakeBool(e.Run(currField, tagExpr)) {
			return false
		}
	}
	return true
}

type orExprNode struct{ exprBackground }

func newOrExprNode() ExprNode { return &orExprNode{} }

func (oe *orExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	for _, e := range [2]ExprNode{oe.leftOperand, oe.rightOperand} {
		if FakeBool(e.Run(currField, tagExpr)) {
			return true
		}
	}
	return false
}
