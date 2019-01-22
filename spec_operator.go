package tagexpr

import "math"

// --------------------------- Operator ---------------------------

type additionExprNode struct{ exprBackground }

func newAdditionExprNode() ExprNode { return &additionExprNode{} }

func (ae *additionExprNode) Eval() interface{} {
	// positive number or Addition
	v0 := ae.leftOperand.Eval()
	v1 := ae.rightOperand.Eval()
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

func (ae *multiplicationExprNode) Eval() interface{} {
	v0, _ := ae.leftOperand.Eval().(float64)
	v1, _ := ae.rightOperand.Eval().(float64)
	return v0 * v1
}

type divisionExprNode struct{ exprBackground }

func newDivisionExprNode() ExprNode { return &divisionExprNode{} }

func (de *divisionExprNode) Eval() interface{} {
	v1, _ := de.rightOperand.Eval().(float64)
	if v1 == 0 {
		return math.NaN()
	}
	v0, _ := de.leftOperand.Eval().(float64)
	return v0 / v1
}

type subtractionExprNode struct{ exprBackground }

func newSubtractionExprNode() ExprNode { return &subtractionExprNode{} }

func (de *subtractionExprNode) Eval() interface{} {
	v0, _ := de.leftOperand.Eval().(float64)
	v1, _ := de.rightOperand.Eval().(float64)
	return v0 - v1
}

type remainderExprNode struct{ exprBackground }

func newRemainderExprNode() ExprNode { return &remainderExprNode{} }

func (re *remainderExprNode) Eval() interface{} {
	v1, _ := re.rightOperand.Eval().(float64)
	if v1 == 0 {
		return math.NaN()
	}
	v0, _ := re.leftOperand.Eval().(float64)
	return float64(int64(v0) % int64(v1))
}

type equalExprNode struct{ exprBackground }

func newEqualExprNode() ExprNode { return &equalExprNode{} }

func (ee *equalExprNode) Eval() interface{} {
	v0 := ee.leftOperand.Eval()
	v1 := ee.rightOperand.Eval()
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

type notEqualExprNode struct{ equalExprNode }

func newNotEqualExprNode() ExprNode { return &notEqualExprNode{} }

func (ne *notEqualExprNode) Eval() interface{} {
	return !ne.equalExprNode.Eval().(bool)
}

type greaterExprNode struct{ exprBackground }

func newGreaterExprNode() ExprNode { return &greaterExprNode{} }

func (ge *greaterExprNode) Eval() interface{} {
	v0 := ge.leftOperand.Eval()
	v1 := ge.rightOperand.Eval()
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

type greaterEqualExprNode struct{ exprBackground }

func newGreaterEqualExprNode() ExprNode { return &greaterEqualExprNode{} }

func (ge *greaterEqualExprNode) Eval() interface{} {
	v0 := ge.leftOperand.Eval()
	v1 := ge.rightOperand.Eval()
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

type lessExprNode struct{ exprBackground }

func newLessExprNode() ExprNode { return &lessExprNode{} }

func (le *lessExprNode) Eval() interface{} {
	v0 := le.leftOperand.Eval()
	v1 := le.rightOperand.Eval()
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

type lessEqualExprNode struct{ exprBackground }

func newLessEqualExprNode() ExprNode { return &lessEqualExprNode{} }

func (le *lessEqualExprNode) Eval() interface{} {
	v0 := le.leftOperand.Eval()
	v1 := le.rightOperand.Eval()
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

type andExprNode struct{ exprBackground }

func newAndExprNode() ExprNode { return &andExprNode{} }

func (ae *andExprNode) Eval() interface{} {
	for _, e := range []ExprNode{ae.leftOperand, ae.rightOperand} {
		switch r := e.Eval().(type) {
		case float64:
			if r == 0 {
				return false
			}
		case string:
			if r == "" {
				return false
			}
		case bool:
			if !r {
				return false
			}
		case nil:
			return false
		default:
			return false
		}
	}
	return true
}

type orExprNode struct{ exprBackground }

func newOrExprNode() ExprNode { return &orExprNode{} }

func (oe *orExprNode) Eval() interface{} {
	for _, e := range []ExprNode{oe.leftOperand, oe.rightOperand} {
		switch r := e.Eval().(type) {
		case float64:
			if r != 0 {
				return true
			}
		case string:
			if r != "" {
				return true
			}
		case bool:
			if r {
				return true
			}
		}
	}
	return false
}
