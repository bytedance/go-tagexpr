package tagexpr

import (
	"fmt"
)

// Expr expression
type Expr struct {
	expr      ExprNode
	varGetter func(string) interface{}
}

// ExprNode expression interface
type ExprNode interface {
	SetParent(ExprNode)
	Parent() ExprNode
	LeftOperand() ExprNode
	RightOperand() ExprNode
	SetLeftOperand(ExprNode)
	SetRightOperand(ExprNode)
	Eval() interface{}
}

// parseExpr parses the expression.
func parseExpr(expr string, varGetter func(string) interface{}) (*Expr, error) {
	e := newGroupExprNode()
	p := &Expr{
		expr:      e,
		varGetter: varGetter,
	}
	s := expr
	_, err := p.parseExprNode(&s, e)
	if err != nil {
		return nil, fmt.Errorf("%q (syntax incorrect): %s", expr, err.Error())
	}
	sortPriority(e.RightOperand())
	err = p.checkSyntax()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Eval calculates the value of expression.
func (p *Expr) Eval() interface{} {
	return p.expr.Eval()
}

func (p *Expr) parseOperand(expr *string) (e ExprNode) {
	if e = p.readLenFnExprNode(expr); e != nil {
		return e
	}
	if e = p.readRegexpFnExprNode(expr); e != nil {
		return e
	}
	if e = p.readSprintfFnExprNode(expr); e != nil {
		return e
	}
	if e = readStringExprNode(expr); e != nil {
		return e
	}
	if e = readDigitalExprNode(expr); e != nil {
		return e
	}
	if e = readBoolExprNode(expr); e != nil {
		return e
	}
	return nil
}

func (*Expr) parseOperator(expr *string) (e ExprNode) {
	s := *expr
	if len(s) < 2 {
		return nil
	}
	defer func() {
		if e != nil && *expr == s {
			*expr = (*expr)[2:]
		}
	}()
	a := s[:2]
	switch a {
	// case "<<":
	// case ">>":
	// case "&^":
	case "||":
		return newOrExprNode()
	case "&&":
		return newAndExprNode()
	case "==":
		return newEqualExprNode()
	case ">=":
		return newGreaterEqualExprNode()
	case "<=":
		return newLessEqualExprNode()
	case "!=":
		return newNotEqualExprNode()
	}
	defer func() {
		if e != nil {
			*expr = (*expr)[1:]
		}
	}()
	switch a[0] {
	// case '&':
	// case '|':
	// case '^':
	case '+':
		return newAdditionExprNode()
	case '-':
		return newSubtractionExprNode()
	case '*':
		return newMultiplicationExprNode()
	case '/':
		return newDivisionExprNode()
	case '%':
		return newRemainderExprNode()
	case '<':
		return newLessExprNode()
	case '>':
		return newGreaterExprNode()
	}
	return nil
}

func (p *Expr) parseExprNode(expr *string, e ExprNode) (ExprNode, error) {
	trimLeftSpace(expr)
	if *expr == "" {
		return nil, nil
	}
	operand, subExprNode := readGroupExprNode(expr)
	if operand != nil {
		_, err := p.parseExprNode(subExprNode, operand)
		if err != nil {
			return nil, err
		}
	} else {
		operand = p.parseOperand(expr)
	}

	if operand == nil {
		return nil, fmt.Errorf("parsing pos: %q", *expr)
	}

	trimLeftSpace(expr)
	operator := p.parseOperator(expr)
	if operator == nil {
		e.SetRightOperand(operand)
		operand.SetParent(e)
		return operand, nil
	}
	if _, ok := e.(*groupExprNode); ok {
		operator.SetLeftOperand(operand)
		operand.SetParent(operator)
		e.SetRightOperand(operator)
		operator.SetParent(e)
	} else {
		e.SetRightOperand(operand)
		operand.SetParent(e)
		operator.SetLeftOperand(e)
		operator.SetParent(e.Parent())
		operator.Parent().SetRightOperand(operator)
		e.SetParent(operator)
	}
	return p.parseExprNode(expr, operator)
}

func (p *Expr) checkSyntax() error {

	return nil
}

/**
 * Priority:
 * () bool string float64
 * * / %
 * + -
 * < <= > >=
 * == !=
 * &&
 * ||
**/

func sortPriority(e ExprNode) {
	if e == nil {
		return
	}
	sortPriority(e.LeftOperand())
	sortPriority(e.RightOperand())
	if getPriority(e) > getPriority(e.LeftOperand()) {
		leftOperandToParent(e)
	}
}

func getPriority(e ExprNode) (i int) {
	// defer func() {
	// 	fmt.Printf("expr:%T %d\n", e, i)
	// }()
	switch e.(type) {
	default: // () bool string float64
		return 7
	case *multiplicationExprNode, *divisionExprNode, *remainderExprNode: // * / %
		return 6
	case *additionExprNode, *subtractionExprNode: // + -
		return 5
	case *lessExprNode, *lessEqualExprNode, *greaterExprNode, *greaterEqualExprNode: // < <= > >=
		return 4
	case *equalExprNode, *notEqualExprNode: // == !=
		return 3
	case *andExprNode: // &&
		return 2
	case *orExprNode: // ||
		return 1
	}
}

func leftOperandToParent(e ExprNode) {
	le := e.LeftOperand()
	if le == nil {
		return
	}
	e.SetLeftOperand(le.RightOperand())
	le.SetRightOperand(e)
	p := e.Parent()
	// if p == nil {
	// 	return
	// }
	if p.LeftOperand() == e {
		p.SetLeftOperand(le)
	} else {
		p.SetRightOperand(le)
	}
	le.SetParent(p)
	e.SetParent(le)
}

type exprBackground struct {
	parent       ExprNode
	leftOperand  ExprNode
	rightOperand ExprNode
}

func (eb *exprBackground) SetParent(e ExprNode) {
	eb.parent = e
}

func (eb *exprBackground) Parent() ExprNode {
	return eb.parent
}

func (eb *exprBackground) LeftOperand() ExprNode {
	return eb.leftOperand
}

func (eb *exprBackground) RightOperand() ExprNode {
	return eb.rightOperand
}

func (eb *exprBackground) SetLeftOperand(left ExprNode) {
	eb.leftOperand = left
}

func (eb *exprBackground) SetRightOperand(right ExprNode) {
	eb.rightOperand = right
}

func (*exprBackground) Eval() interface{} { return nil }
