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
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/henrylee2cn/goutil/errors"
)

// --------------------------- Custom function ---------------------------

var funcList = map[string]func(p *Expr, expr *string) ExprNode{}

// RegSimpleFunc registers simple function expression.
// NOTE:
//  example: len($) or len() to returns the v's length;
//  If @force=true, allow to cover the existed same @funcName;
//  The go number types always are float64;
//  The go string types always are string.
func RegSimpleFunc(funcName string, fn func(v interface{}) interface{}, force ...bool) error {
	if len(force) == 0 || !force[0] {
		_, ok := funcList[funcName]
		if ok {
			return errors.Errorf("duplicate registration expression function: %s", funcName)
		}
	}
	funcList[funcName] = newSimpleFunc(funcName, fn)
	return nil
}

func newSimpleFunc(funcName string, fn func(interface{}) interface{}) func(*Expr, *string) ExprNode {
	return func(p *Expr, expr *string) ExprNode {
		if !strings.HasPrefix(*expr, funcName+"(") {
			return nil
		}
		*expr = (*expr)[len(funcName):]
		lastStr := *expr
		s := strings.TrimLeftFunc((*expr)[1:], unicode.IsSpace)
		if strings.HasPrefix(s, ")") {
			*expr = "($" + s
		}
		operand, subExprNode := readGroupExprNode(expr)
		if operand == nil {
			return nil
		}
		_, err := p.parseExprNode(subExprNode, operand)
		if err != nil {
			*expr = lastStr
			return nil
		}
		e := &simpleFuncExprNode{fn: fn}
		e.SetRightOperand(operand)
		return e
	}
}

type simpleFuncExprNode struct {
	exprBackground
	fn func(v interface{}) interface{}
}

func (sfn *simpleFuncExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	return sfn.fn(sfn.rightOperand.Run(currField, tagExpr))
}

// --------------------------- Built-in function ---------------------------
func init() {
	funcList["regexp"] = readRegexpFuncExprNode
	funcList["sprintf"] = readSprintfFuncExprNode
	err := RegSimpleFunc("len", func(param interface{}) interface{} {
		switch v := param.(type) {
		case string:
			return float64(len(v))
		case float64, bool:
			return nil
		}
		defer func() { recover() }()
		v := reflect.ValueOf(param)
		return float64(v.Len())
	})
	if err != nil {
		panic(err)
	}
}

type regexpFuncExprNode struct {
	exprBackground
	re *regexp.Regexp
}

func readRegexpFuncExprNode(p *Expr, expr *string) ExprNode {
	if !strings.HasPrefix(*expr, "regexp(") {
		return nil
	}
	*expr = (*expr)[6:]
	lastStr := *expr
	subExprNode := readPairedSymbol(expr, '(', ')')
	if subExprNode == nil {
		return nil
	}
	s := readPairedSymbol(trimLeftSpace(subExprNode), '\'', '\'')
	if s == nil {
		*expr = lastStr
		return nil
	}
	rege, err := regexp.Compile(*s)
	if err != nil {
		*expr = lastStr
		return nil
	}
	operand := newGroupExprNode()
	trimLeftSpace(subExprNode)
	if strings.HasPrefix(*subExprNode, ",") {
		*subExprNode = (*subExprNode)[1:]
		_, err = p.parseExprNode(trimLeftSpace(subExprNode), operand)
		if err != nil {
			*expr = lastStr
			return nil
		}
	} else {
		var currFieldVal = "$"
		p.parseExprNode(&currFieldVal, operand)
	}
	trimLeftSpace(subExprNode)
	if *subExprNode != "" {
		*expr = lastStr
		return nil
	}
	e := &regexpFuncExprNode{
		re: rege,
	}
	e.SetRightOperand(operand)
	return e
}

func (re *regexpFuncExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	param := re.rightOperand.Run(currField, tagExpr)
	switch v := param.(type) {
	case string:
		return re.re.MatchString(v)
	case float64, bool:
		return nil
	}
	v := reflect.ValueOf(param)
	if v.Kind() == reflect.String {
		return re.re.MatchString(v.String())
	}
	return nil
}

type sprintfFuncExprNode struct {
	exprBackground
	format string
	args   []ExprNode
}

func readSprintfFuncExprNode(p *Expr, expr *string) ExprNode {
	if !strings.HasPrefix(*expr, "sprintf(") {
		return nil
	}
	*expr = (*expr)[7:]
	lastStr := *expr
	subExprNode := readPairedSymbol(expr, '(', ')')
	if subExprNode == nil {
		return nil
	}
	format := readPairedSymbol(trimLeftSpace(subExprNode), '\'', '\'')
	if format == nil {
		*expr = lastStr
		return nil
	}
	e := &sprintfFuncExprNode{
		format: *format,
	}
	for {
		trimLeftSpace(subExprNode)
		if len(*subExprNode) == 0 {
			return e
		}
		if strings.HasPrefix(*subExprNode, ",") {
			*subExprNode = (*subExprNode)[1:]
			operand := newGroupExprNode()
			_, err := p.parseExprNode(trimLeftSpace(subExprNode), operand)
			if err != nil {
				*expr = lastStr
				return nil
			}
			sortPriority(operand.RightOperand())
			e.args = append(e.args, operand)
		} else {
			*expr = lastStr
			return nil
		}
	}
}

func (se *sprintfFuncExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	var args []interface{}
	if n := len(se.args); n > 0 {
		args = make([]interface{}, n)
		for i, e := range se.args {
			args[i] = e.Run(currField, tagExpr)
		}
	}
	return fmt.Sprintf(se.format, args...)
}
