package tagexpr

import (
	"fmt"
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
	sortPriority(e.RightOperand())
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

func (i *Interpreter) parseOperand(expr *string) (e Expr) {
	if e = i.readLenFnExpr(expr); e != nil {
		return e
	}
	if e = readStringExpr(expr); e != nil {
		return e
	}
	if e = readDigitalExpr(expr); e != nil {
		return e
	}
	if e = readBoolExpr(expr); e != nil {
		return e
	}
	return nil
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
	if *expr == "" {
		return nil, nil
	}
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

func sortPriority(e Expr) {
	if e == nil {
		return
	}
	sortPriority(e.LeftOperand())
	sortPriority(e.RightOperand())
	if getPriority(e) > getPriority(e.LeftOperand()) {
		leftOperandToParent(e)
	}
}

func getPriority(e Expr) (i int) {
	// defer func() {
	// 	fmt.Printf("expr:%T %d\n", e, i)
	// }()
	switch e.(type) {
	default: // () bool string float64
		return 7
	case *multiplicationExpr, *divisionExpr, *remainderExpr: // * / %
		return 6
	case *additionExpr, *subtractionExpr: // + -
		return 5
	case *lessExpr, *lessEqualExpr, *greaterExpr, *greaterEqualExpr: // < <= > >=
		return 4
	case *equalExpr, *notEqualExpr: // == !=
		return 3
	case *andExpr: // &&
		return 2
	case *orExpr: // ||
		return 1
	}
}

func leftOperandToParent(e Expr) {
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
