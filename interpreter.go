package tagexpr

import (
	"strings"
	"unicode"
)

type Interpreter struct {
	expr Expr
}

// New parses the expression and creates an interpreter.
func New(expr string) (*Interpreter, error) {
	e := newGroupExpr()
	s := expr
	_, err := parseExpr(&s, e)
	if err != nil {
		return nil, err
	}
	return &Interpreter{expr: e}, nil
}

// Run calculates the value of expression.
func (i *Interpreter) Run() interface{} {
	return i.expr.Calculate()
}

func parseOperand(expr *string) (e Expr) {
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

func parseOperator(expr *string) (e Expr) {
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
	case '/':
	case '%':
	case '<':
	case '>':
	case '=':
	default:
	}
	return nil
}

func parseExpr(expr *string, e Expr) (Expr, error) {
	trimLeftSpace(expr)
	operand, subExpr := readGroupExpr(expr)
	if operand != nil {
		_ = subExpr
	}
	trimLeftSpace(expr)
	operand = parseOperand(expr)
	trimLeftSpace(expr)
	operator := parseOperator(expr)
	if operator == nil {
		e.SetRightOperand(operand)
		operand.SetParent(e)
		return operand, nil
	}
	operator.SetLeftOperand(operand)
	e.SetRightOperand(operator)
	operator.SetParent(e)
	return parseExpr(expr, operator)
}

func trimLeftSpace(p *string) *string {
	*p = strings.TrimLeftFunc(*p, unicode.IsSpace)
	return p
}
