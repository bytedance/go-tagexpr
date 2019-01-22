package tagexpr

import (
	"fmt"
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
		return nil, fmt.Errorf("%q (syntax incorrect): %s", expr, err.Error())
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
	e = readBoolExpr(expr)
	if e != nil {
		return e
	}
	e = readStringExpr(expr)
	if e != nil {
		return e
	}
	e = readDigitalExpr(expr)
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
		return newOrExpr()
	case "&&":
		return newAndExpr()
	case "==":
		return newEqualExpr()
	case ">=":
		return newGreaterEqualExpr()
	case "<=":
		return newLessEqualExpr()
	case "!=":
		return newNotEqualExpr()
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
		return newAdditionExpr()
	case '-':
		return newSubtractionExpr()
	case '*':
		return newMultiplicationExpr()
	case '/':
		return newDivisionExpr()
	case '%':
		return newRemainderExpr()
	case '<':
		return newLessExpr()
	case '>':
		return newGreaterExpr()
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

	if operand == nil {
		return nil, fmt.Errorf("expect operand but got: %q", *expr)
	}

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

func (i *Interpreter) checkSyntax() error {
	return nil
}

func trimLeftSpace(p *string) *string {
	*p = strings.TrimLeftFunc(*p, unicode.IsSpace)
	return p
}
