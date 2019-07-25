// Package validator is a powerful validator that supports struct tag expression.
//
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
package validator

import (
	"errors"
	"io"
	"strings"
	_ "unsafe"

	tagexpr "github.com/bytedance/go-tagexpr"
)

const (
	// MatchExprName the name of the expression used for validation
	MatchExprName = tagexpr.DefaultExprName
	// ErrMsgExprName the name of the expression used to specify the message
	// returned when validation failed
	ErrMsgExprName = "msg"
)

// Validator struct fields validator
type Validator struct {
	vm         *tagexpr.VM
	errFactory func(failPath, msg string) error
}

// New creates a struct fields validator.
func New(tagName string) *Validator {
	v := &Validator{
		vm:         tagexpr.New(tagName),
		errFactory: defaultErrorFactory,
	}
	return v
}

// VM returns the struct tag expression interpreter.
func (v *Validator) VM() *tagexpr.VM {
	return v.vm
}

// Validate validates whether the fields of value is valid.
func (v *Validator) Validate(value interface{}, checkAll ...bool) error {
	var all bool
	if len(checkAll) > 0 {
		all = checkAll[0]
	}
	var errs []error
	v.vm.RunAny(value, func(te *tagexpr.TagExpr, err error) error {
		if err != nil {
			errs = append(errs, err)
			if !all {
				return io.EOF
			}
		}
		var errSelector, errPath string
		te.Range(func(path string, es tagexpr.ExprSelector, eval func() interface{}) error {
			if strings.Contains(path, tagexpr.ExprNameSeparator) {
				return nil
			}
			valid := tagexpr.FakeBool(eval())
			if valid {
				return nil
			}
			errSelector = es.String()
			errPath = path
			if all {
				return nil
			}
			return io.EOF
		})
		if errSelector == "" {
			return nil
		}
		errs = append(errs, v.errFactory(
			errPath,
			te.EvalString(errSelector+tagexpr.ExprNameSeparator+ErrMsgExprName),
		))
		if all {
			return nil
		}
		return io.EOF
	})
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		var errStr string
		for _, e := range errs {
			errStr += e.Error() + "\n"
		}
		return errors.New(errStr[:len(errStr)-1])
	}
}

// SetErrorFactory customizes the factory of validation error.
// NOTE:
//  If errFactory==nil, the default is used
func (v *Validator) SetErrorFactory(errFactory func(failPath, msg string) error) *Validator {
	if errFactory == nil {
		errFactory = defaultErrorFactory
	}
	v.errFactory = errFactory
	return v
}

// Error validate error
type Error struct {
	FailPath, Msg string
}

// Error implements error interface.
func (e *Error) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "invalid parameter: " + e.FailPath
}

//go:linkname defaultErrorFactory validator.defaultErrorFactory
//go:nosplit
func defaultErrorFactory(failPath, msg string) error {
	return &Error{
		FailPath: failPath,
		Msg:      msg,
	}
}
