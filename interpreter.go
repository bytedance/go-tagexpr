package tagexpr

import (
	"strings"
	"unicode"
)

// Interpreter expression VM
type Interpreter struct {
	expr Expr
}

// New parses the expression and creates an interpreter.
func New(expr string) (*Interpreter, error) {
	e := newGroupExpr()
	i := &Interpreter{expr: e}
	s := expr
	_, err := i.parseExpr(&s, e)
	if err != nil {
		return nil, err
	}
	err = i.checkSyntax()
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Run calculates the value of expression.
func (i *Interpreter) Run() interface{} {
	return i.expr.Calculate()
}

func (*Interpreter) parseOperand(expr *string) (e Expr) {
	e = readStringExpr(expr)
	if e != nil {
		return e
	}
	e = readDigitalExpr(expr)
	if e != nil {
		return e
	}
	e = readBoolExpr(expr)
	if e != nil {
		return e
	}
	return e
}

func (*Interpreter) parseOperator(expr *string) (e Expr) {
	s := *expr
	if len(s) < 2 {
		return nil
	}
	a := s[:2]
	switch a {
	case "<<":
	case ">>":
	case "&^":
	case "||":
	case "&&":
	case "==":
	case ">=":
	case "<=":
	case "!=":
	default:
	}
	defer func() {
		if e != nil {
			*expr = (*expr)[1:]
		}
	}()
	switch a[0] {
	case '&':
	case '|':
	case '^':
	case '+':
		return newAdditionExpr()
	case '-':
	case '*':
		return newMultiplicationExprExpr()
	case '/':
	case '%':
	case '<':
	case '>':
	case '=':
	default:
	}
	return nil
}

func (i *Interpreter) parseExpr(expr *string, e Expr) (Expr, error) {
	trimLeftSpace(expr)
	operand, subExpr := readGroupExpr(expr)
	if operand != nil {
		_, err := i.parseExpr(subExpr, operand)
		if err != nil {
			return nil, err
		}
	} else {
		operand = i.parseOperand(expr)
	}

	// ?

	trimLeftSpace(expr)
	operator := i.parseOperator(expr)
	if operator == nil {
		e.SetRightOperand(operand)
		operand.SetParent(e)
		return operand, nil
	}
	if _, ok := e.(*groupExpr); ok {
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
	return i.parseExpr(expr, operator)
}

// func (i *Interpreter) parseExpr(expr *string, e Expr) (Expr, error) {
// 	trimLeftSpace(expr)
// 	operand, subExpr := readGroupExpr(expr)
// 	if operand != nil {
// 		_ = subExpr
// 	}
// 	trimLeftSpace(expr)
// 	operand = i.parseOperand(expr)
// 	trimLeftSpace(expr)
// 	operator := i.parseOperator(expr)
// 	if operator == nil {
// 		e.SetRightOperand(operand)
// 		operand.SetParent(e)
// 		return operand, nil
// 	}
// 	operator.SetLeftOperand(operand)
// 	e.SetRightOperand(operator)
// 	operator.SetParent(e)
// 	return i.parseExpr(expr, operator)
// }

func (i *Interpreter) checkSyntax() error {
	return nil
}

func trimLeftSpace(p *string) *string {
	*p = strings.TrimLeftFunc(*p, unicode.IsSpace)
	return p
}
