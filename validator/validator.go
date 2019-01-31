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
	tagexpr "github.com/bytedance/go-tagexpr"
)

const matchExprName = "@"
const errMsgExprName = "msg"

// Validator struct fields validator
type Validator struct {
	vm         *tagexpr.VM
	errFactory func(fieldSelector, msg string) error
}

// New creates a struct fields validator.
func New(tagName string) *Validator {
	v := &Validator{
		vm:         tagexpr.New(tagName),
		errFactory: defaultErrorFactory,
	}
	return v
}

// Validate validates whether the fields of structPtr is valid.
func (v *Validator) Validate(structPtr interface{}) error {
	expr, err := v.vm.Run(structPtr)
	if err != nil {
		return err
	}
	var errSelector string
	expr.Range(func(selector string, eval func() interface{}) bool {
		if !isMatchSelector(selector) {
			return true
		}
		valid, _ := eval().(bool)
		if !valid {
			errSelector = selector
		}
		return valid
	})
	if errSelector == "" {
		return nil
	}
	errMsg := expr.EvalString(errSelector + errMsgExprName)
	return v.errFactory(errSelector[:len(errSelector)-1], errMsg)
}

// SetErrorFactory customizes the factory of validation error.
func (v *Validator) SetErrorFactory(errFactory func(fieldSelector, msg string) error) *Validator {
	v.errFactory = errFactory
	return v
}

func isMatchSelector(selector string) bool {
	n := len(selector)
	return n > 1 && selector[n-1] == '@' && selector[n-2] != '@'
}

// Error validate error
type Error struct {
	FieldSelector, Msg string
}

// Error implements error interface.
func (e *Error) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "Invalid parameter: " + e.FieldSelector
}

func defaultErrorFactory(fieldSelector, msg string) error {
	return &Error{
		FieldSelector: fieldSelector,
		Msg:           msg,
	}
}
