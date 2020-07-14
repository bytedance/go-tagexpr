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

	"github.com/henrylee2cn/goutil/errors"
)

// --------------------------- Custom function ---------------------------

var funcList = map[string]func(p *Expr, expr *string) ExprNode{}

// RegFunc registers function expression.
// NOTE:
//  example: len($), regexp("\\d") or regexp("\\d",$);
//  If @force=true, allow to cover the existed same @funcName;
//  The go number types always are float64;
//  The go string types always are string.
func RegFunc(funcName string, fn func(...interface{}) interface{}, force ...bool) error {
	if len(force) == 0 || !force[0] {
		_, ok := funcList[funcName]
		if ok {
			return errors.Errorf("duplicate registration expression function: %s", funcName)
		}
	}
	funcList[funcName] = newFunc(funcName, fn)
	return nil
}

func newFunc(funcName string, fn func(...interface{}) interface{}) func(*Expr, *string) ExprNode {
	prefix := funcName + "("
	length := len(funcName)
	return func(p *Expr, expr *string) ExprNode {
		last, boolOpposite := getBoolOpposite(expr)
		if !strings.HasPrefix(last, prefix) {
			return nil
		}
		*expr = last[length:]
		lastStr := *expr
		subExprNode := readPairedSymbol(expr, '(', ')')
		if subExprNode == nil {
			return nil
		}
		f := &funcExprNode{
			fn:           fn,
			boolOpposite: boolOpposite,
		}
		*subExprNode = "," + *subExprNode
		for {
			if strings.HasPrefix(*subExprNode, ",") {
				*subExprNode = (*subExprNode)[1:]
				operand := newGroupExprNode()
				_, err := p.parseExprNode(trimLeftSpace(subExprNode), operand)
				if err != nil {
					*expr = lastStr
					return nil
				}
				sortPriority(operand.RightOperand())
				f.args = append(f.args, operand)
			} else {
				*expr = lastStr
				return nil
			}
			trimLeftSpace(subExprNode)
			if len(*subExprNode) == 0 {
				return f
			}
		}
	}
}

type funcExprNode struct {
	exprBackground
	args         []ExprNode
	fn           func(...interface{}) interface{}
	boolOpposite *bool
}

func (f *funcExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	var args []interface{}
	if n := len(f.args); n > 0 {
		args = make([]interface{}, n)
		for k, v := range f.args {
			args[k] = v.Run(currField, tagExpr)
		}
	}
	return realValue(f.fn(args...), f.boolOpposite)
}

// --------------------------- Built-in function ---------------------------
func init() {
	funcList["regexp"] = readRegexpFuncExprNode
	funcList["sprintf"] = readSprintfFuncExprNode
	err := RegFunc("len", func(args ...interface{}) (n interface{}) {
		if len(args) != 1 {
			return 0
		}
		v := args[0]
		switch e := v.(type) {
		case string:
			return float64(len(e))
		case float64, bool, nil:
			return 0
		}
		defer func() {
			if recover() != nil {
				n = 0
			}
		}()
		return float64(reflect.ValueOf(v).Len())
	}, true)
	if err != nil {
		panic(err)
	}
	err = RegFunc("mblen", func(args ...interface{}) (n interface{}) {
		if len(args) != 1 {
			return 0
		}
		v := args[0]
		switch e := v.(type) {
		case string:
			return float64(len([]rune(e)))
		case float64, bool, nil:
			return 0
		}
		defer func() {
			if recover() != nil {
				n = 0
			}
		}()
		return float64(reflect.ValueOf(v).Len())
	}, true)
	if err != nil {
		panic(err)
	}
}

type regexpFuncExprNode struct {
	exprBackground
	re           *regexp.Regexp
	boolOpposite bool
}

func readRegexpFuncExprNode(p *Expr, expr *string) ExprNode {
	last, boolOpposite := getBoolOpposite(expr)
	if !strings.HasPrefix(last, "regexp(") {
		return nil
	}
	*expr = last[6:]
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
	if boolOpposite != nil {
		e.boolOpposite = *boolOpposite
	}
	e.SetRightOperand(operand)
	return e
}

func (re *regexpFuncExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	param := re.rightOperand.Run(currField, tagExpr)
	switch v := param.(type) {
	case string:
		bol := re.re.MatchString(v)
		if re.boolOpposite {
			return !bol
		}
		return bol
	case float64, bool:
		return false
	}
	v := reflect.ValueOf(param)
	if v.Kind() == reflect.String {
		bol := re.re.MatchString(v.String())
		if re.boolOpposite {
			return !bol
		}
		return bol
	}
	return false
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
